package platform

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestReadJSONFile(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "test.json")
	_ = os.WriteFile(f, []byte(`{"name":"alice","age":30}`), 0644)

	var data struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}
	if err := ReadJSONFile(f, &data); err != nil {
		t.Fatalf("ReadJSONFile() error = %v", err)
	}
	if data.Name != "alice" {
		t.Errorf("Name = %q, want %q", data.Name, "alice")
	}
	if data.Age != 30 {
		t.Errorf("Age = %d, want %d", data.Age, 30)
	}
}

func TestReadJSONFile_MissingFile(t *testing.T) {
	err := ReadJSONFile("/nonexistent/path.json", &struct{}{})
	if err == nil {
		t.Error("ReadJSONFile() expected error for missing file")
	}
}

func TestReadJSONFile_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "bad.json")
	_ = os.WriteFile(f, []byte(`not json`), 0644)

	err := ReadJSONFile(f, &struct{}{})
	if err == nil {
		t.Error("ReadJSONFile() expected error for invalid JSON")
	}
}

func TestWriteJSONFile(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "out.json")

	data := map[string]string{"key": "value"}
	if err := WriteJSONFile(f, data); err != nil {
		t.Fatalf("WriteJSONFile() error = %v", err)
	}

	got, err := os.ReadFile(f)
	if err != nil {
		t.Fatalf("reading file: %v", err)
	}

	want := "{\n  \"key\": \"value\"\n}\n"
	if string(got) != want {
		t.Errorf("WriteJSONFile() content = %q, want %q", got, want)
	}
}

func TestWriteJSONFile_Permissions(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "perms.json")

	_ = WriteJSONFile(f, map[string]string{})

	info, err := os.Stat(f)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if info.Mode().Perm() != 0644 {
		t.Errorf("permissions = %o, want 0644", info.Mode().Perm())
	}
}

func TestWriteJSONFile_NestedStruct(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "nested.json")

	type Inner struct {
		Value int `json:"value"`
	}
	type Outer struct {
		Name  string `json:"name"`
		Inner Inner  `json:"inner"`
	}

	data := Outer{Name: "test", Inner: Inner{Value: 42}}
	if err := WriteJSONFile(f, data); err != nil {
		t.Fatalf("WriteJSONFile() error = %v", err)
	}

	var loaded Outer
	if err := ReadJSONFile(f, &loaded); err != nil {
		t.Fatalf("ReadJSONFile() error = %v", err)
	}
	if loaded.Name != "test" || loaded.Inner.Value != 42 {
		t.Errorf("roundtrip = %+v, want {test {42}}", loaded)
	}
}

func TestReadWriteJSONFile_Roundtrip(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "roundtrip.json")

	type Config struct {
		Host string   `json:"host"`
		Port int      `json:"port"`
		Tags []string `json:"tags"`
	}

	original := Config{Host: "localhost", Port: 8080, Tags: []string{"a", "b"}}
	if err := WriteJSONFile(f, original); err != nil {
		t.Fatalf("WriteJSONFile() error = %v", err)
	}

	var loaded Config
	if err := ReadJSONFile(f, &loaded); err != nil {
		t.Fatalf("ReadJSONFile() error = %v", err)
	}

	if loaded.Host != original.Host || loaded.Port != original.Port {
		t.Errorf("roundtrip mismatch: got %+v, want %+v", loaded, original)
	}
	if len(loaded.Tags) != len(original.Tags) {
		t.Fatalf("tags length: got %d, want %d", len(loaded.Tags), len(original.Tags))
	}
	for i := range original.Tags {
		if loaded.Tags[i] != original.Tags[i] {
			t.Errorf("tags[%d] = %q, want %q", i, loaded.Tags[i], original.Tags[i])
		}
	}
}

func TestReadJSONFileRaw(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "raw.json")
	_ = os.WriteFile(f, []byte(`{"name":"bob","nested":{"x":1}}`), 0644)

	m, err := ReadJSONFileRaw(f)
	if err != nil {
		t.Fatalf("ReadJSONFileRaw() error = %v", err)
	}

	if len(m) != 2 {
		t.Fatalf("returned %d keys, want 2", len(m))
	}

	var name string
	if err := json.Unmarshal(m["name"], &name); err != nil {
		t.Fatalf("unmarshaling name: %v", err)
	}
	if name != "bob" {
		t.Errorf("name = %q, want %q", name, "bob")
	}

	if string(m["nested"]) != `{"x":1}` {
		t.Errorf("nested = %s, want %s", m["nested"], `{"x":1}`)
	}
}

func TestReadJSONFileRaw_PreservesRawValues(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "raw.json")
	_ = os.WriteFile(f, []byte(`{"arr":[1,2,3],"bool":true,"null":null}`), 0644)

	m, err := ReadJSONFileRaw(f)
	if err != nil {
		t.Fatalf("ReadJSONFileRaw() error = %v", err)
	}

	if string(m["arr"]) != "[1,2,3]" {
		t.Errorf("arr = %s, want [1,2,3]", m["arr"])
	}
	if string(m["bool"]) != "true" {
		t.Errorf("bool = %s, want true", m["bool"])
	}
	if string(m["null"]) != "null" {
		t.Errorf("null = %s, want null", m["null"])
	}
}

func TestReadJSONFileRaw_MissingFile(t *testing.T) {
	_, err := ReadJSONFileRaw("/nonexistent/path.json")
	if err == nil {
		t.Error("ReadJSONFileRaw() expected error for missing file")
	}
}

func TestReadJSONFileRaw_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "bad.json")
	_ = os.WriteFile(f, []byte(`{invalid`), 0644)

	_, err := ReadJSONFileRaw(f)
	if err == nil {
		t.Error("ReadJSONFileRaw() expected error for invalid JSON")
	}
}
