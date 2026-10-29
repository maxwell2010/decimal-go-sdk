// Compile official protobuf schemas without rewriting fields or custom options.
package main

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/bufbuild/protocompile"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

func main() {
	root := flag.String("node", "", "pinned node snapshot")
	sdk := flag.String("sdk", "", "Cosmos SDK snapshot root")
	flag.Parse()
	head, e := exec.Command("git", "-C", *root, "rev-parse", "HEAD").Output()
	if e != nil || strings.TrimSpace(string(head)) != "9e6c6d718d662083c4a524376a66d2c50bd4bc77" {
		panic("node snapshot must match pinned commit")
	}
	if e = exec.Command("git", "-C", *root, "diff", "--quiet", "HEAD", "--", "proto/custom", "proto/third_party").Run(); e != nil {
		panic("dirty node protobuf inputs")
	}
	moduleJSON, e := exec.Command("go", "mod", "download", "-json", "github.com/cosmos/cosmos-sdk@v0.46.16").Output()
	if e != nil {
		panic(e)
	}
	var sdkModule struct{ Dir, Zip, Sum, GoModSum string }
	if e = json.Unmarshal(moduleJSON, &sdkModule); e != nil {
		panic(e)
	}
	if filepath.Clean(*sdk) != filepath.Clean(sdkModule.Dir) {
		panic("sdk root must be verified Cosmos SDK v0.46.16 module cache")
	}
	archive, e := zip.OpenReader(sdkModule.Zip)
	if e != nil {
		panic(e)
	}
	defer archive.Close()
	zipFiles := map[string]*zip.File{}
	for _, f := range archive.File {
		zipFiles[f.Name] = f
	}
	paths := []string{}
	filepath.WalkDir(filepath.Join(*root, "proto/custom/decimal"), func(p string, d os.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if !d.IsDir() && (d.Name() == "tx.proto" || d.Name() == "query.proto") {
			r, _ := filepath.Rel(filepath.Join(*root, "proto/custom"), p)
			paths = append(paths, filepath.ToSlash(r))
		}
		return nil
	})
	paths = append(paths, "cosmos/upgrade/v1beta1/tx.proto", "cosmos/tx/v1beta1/tx.proto", "ethermint/crypto/v1/ethsecp256k1/keys.proto")
	paths = append(paths, "cosmos/tx/v1beta1/service.proto", "cosmos/auth/v1beta1/query.proto", "cosmos/bank/v1beta1/query.proto")
	c := protocompile.Compiler{Resolver: protocompile.WithStandardImports(&protocompile.SourceResolver{ImportPaths: []string{filepath.Join(*root, "proto/custom"), filepath.Join(*root, "proto/third_party"), filepath.Join(*sdk, "proto")}})}
	files, e := c.Compile(context.Background(), paths...)
	if e != nil {
		panic(e)
	}
	set := &descriptorpb.FileDescriptorSet{}
	seen := map[string]bool{}
	var add func(protoreflect.FileDescriptor)
	add = func(f protoreflect.FileDescriptor) {
		if seen[f.Path()] {
			return
		}
		seen[f.Path()] = true
		for i := 0; i < f.Imports().Len(); i++ {
			add(f.Imports().Get(i).FileDescriptor)
		}
		set.File = append(set.File, protodesc.ToFileDescriptorProto(f))
	}
	for _, f := range files {
		add(f)
	}
	sort.Slice(set.File, func(i, j int) bool { return set.File[i].GetName() < set.File[j].GetName() })
	b, e := proto.MarshalOptions{Deterministic: true}.Marshal(set)
	if e != nil {
		panic(e)
	}
	manifest := map[string]any{"node_commit": strings.TrimSpace(string(head)), "cosmos_sdk_version": "v0.46.16", "cosmos_sdk_sum": sdkModule.Sum, "cosmos_sdk_go_mod_sum": sdkModule.GoModSum, "descriptor_sha256": fmt.Sprintf("%x", sha256.Sum256(b))}
	inputs := []map[string]string{}
	for _, f := range set.File {
		found := false
		for _, source := range []struct{ label, root string }{{"node-custom", filepath.Join(*root, "proto/custom")}, {"node-vendored", filepath.Join(*root, "proto/third_party")}, {"cosmos-sdk-v0.46.16", filepath.Join(*sdk, "proto")}} {
			path := filepath.Join(source.root, f.GetName())
			raw, e := os.ReadFile(path)
			if e != nil {
				continue
			}
			if source.label == "cosmos-sdk-v0.46.16" {
				entry := zipFiles["github.com/cosmos/cosmos-sdk@v0.46.16/proto/"+f.GetName()]
				if entry == nil {
					panic("SDK proto absent from module archive")
				}
				reader, e := entry.Open()
				if e != nil {
					panic(e)
				}
				original, e := io.ReadAll(reader)
				reader.Close()
				if e != nil || !bytes.Equal(raw, original) {
					panic("modified SDK protobuf file")
				}
			} else {
				rel, _ := filepath.Rel(*root, path)
				if e = exec.Command("git", "-C", *root, "ls-files", "--error-unmatch", filepath.ToSlash(rel)).Run(); e != nil {
					panic("untracked node protobuf input")
				}
			}
			inputs = append(inputs, map[string]string{"source": source.label, "path": f.GetName(), "sha256": fmt.Sprintf("%x", sha256.Sum256(raw))})
			found = true
			break
		}
		if !found {
			raw, _ := proto.MarshalOptions{Deterministic: true}.Marshal(f)
			inputs = append(inputs, map[string]string{"source": "protocompile-v0.14.1-standard-import", "path": f.GetName(), "descriptor_sha256": fmt.Sprintf("%x", sha256.Sum256(raw))})
		}
	}
	manifest["inputs"] = inputs
	if e = os.WriteFile("schema/decimal.pb", b, 0644); e != nil {
		panic(e)
	}
	proof, e := json.MarshalIndent(manifest, "", "  ")
	if e != nil {
		panic(e)
	}
	if e = os.WriteFile("schema/sources.json", append(proof, '\n'), 0644); e != nil {
		panic(e)
	}
	constructors := "// Code generated from pinned official descriptors; DO NOT EDIT.\npackage cosmos\nimport \"google.golang.org/protobuf/types/dynamicpb\"\n"
	inventory := []string{}
	for _, f := range files {
		service := f.Services().ByName("Msg")
		if service == nil {
			continue
		}
		for i := 0; i < service.Methods().Len(); i++ {
			name := string(service.Methods().Get(i).Input().FullName())
			inventory = append(inventory, "/"+name)
			parts := strings.Split(name, ".")
			constructors += fmt.Sprintf("func New%s()*dynamicpb.Message{return message(%q)}\n", parts[len(parts)-1], name)
		}
	}
	sort.Strings(inventory)
	raw, e := json.MarshalIndent(inventory, "", "  ")
	if e != nil {
		panic(e)
	}
	if e = os.WriteFile("schema/inventory.json", append(raw, '\n'), 0644); e != nil {
		panic(e)
	}
	if e = os.WriteFile("messages_generated.go", []byte(constructors), 0644); e != nil {
		panic(e)
	}
}
