package tssd

import (
	"errors"
	"testing"
)

func TestFragmentUnmarshalSuccess(t *testing.T) {
	payload := []byte("hello fragment")
	data, expectedChecksum := buildFragmentBytes(t, payload, false)

	//we add someting in head, which should drop by TSSD
	data = append(append(make([]byte, 0, 1024), []byte("something")...), data...)

	var frag Fragment
	remaining, err := (&frag).Unmarshal(append(data, []byte("extra")...))
	if err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if string(remaining) != "extra" {
		t.Fatalf("expected remaining bytes %q, got %q", "extra", string(remaining))
	}

	if string(frag.Header.Magic[:]) != MAGIC {
		t.Fatalf("expected magic %q, got %q", MAGIC, frag.Header.Magic)
	}
	if frag.Header.Version[0] != TSSD_VERSION_MINOR || frag.Header.Version[1] != TSSD_VERSION_MAJOR {
		t.Fatalf("unexpected version bytes: %v", frag.Header.Version)
	}
	if frag.Schema.Fragment != 1 {
		t.Fatalf("expected fragment id 1, got %d", frag.Schema.Fragment)
	}
	if frag.Schema.Hash != "hash" || frag.Schema.TID != "tid" || frag.Schema.Extent != "extent" {
		t.Fatalf("unexpected schema: %+v", frag.Schema)
	}
	if string(frag.tdata) != string(payload) {
		t.Fatalf("expected payload %q, got %q", payload, frag.tdata)
	}

	if string(frag.Checksum) != string(expectedChecksum) {
		t.Fatalf("expected checksum %q, got %q", expectedChecksum, frag.Checksum)
	}
}

func TestFragmentUnmarshalRejectsShortInput(t *testing.T) {
	var frag Fragment
	_, err := frag.Unmarshal([]byte(MAGIC))
	if !errors.Is(err, ErrorInSufficientData) {
		t.Fatalf("expected ErrorInSufficientData, got %v", err)
	}
}

func TestFragmentUnmarshalRejectsInvalidMagic(t *testing.T) {
	data, _ := buildFragmentBytes(t, []byte("payload"), false)
	data[0] = 'X'

	var frag Fragment
	_, err := frag.Unmarshal(data)
	if !errors.Is(err, ErrorInvalidTSSDData) {
		t.Fatalf("expected ErrorInvalidTSSDData, got %v", err)
	}
}

func TestFragmentUnmarshalRejectsChecksumMismatch(t *testing.T) {
	data, _ := buildFragmentBytes(t, []byte("payload"), false)
	data[len(data)-1] ^= 1

	var frag Fragment
	_, err := frag.Unmarshal(data)
	if !errors.Is(err, ErrorTSSDDataChecksumFailure) {
		t.Fatalf("expected ErrorTSSDDataChecksumFailure, got %v", err)
	}
}

func TestFragmentUnmarshalDisableChecksum(t *testing.T) {
	data, _ := buildFragmentBytes(t, []byte("payload"), true)
	data[len(data)-9] ^= 1

	var frag Fragment
	_, err := frag.Unmarshal(data)
	if err != nil {
		t.Fatalf("disableChecksum but got ErrorTSSDDataChecksumFailure")
	}
}

func buildFragmentBytes(t *testing.T, payload []byte, disableChecksum bool) ([]byte, []byte) {
	t.Helper()

	buf := &Buffer{MTU: 4096}
	buf.Append([]byte(MAGIC))
	buf.Append([]byte{TSSD_VERSION_MINOR, TSSD_VERSION_MAJOR, Tschema})

	schema := Schema{Fragment: 1, Hash: "hash", TID: "tid", Extent: "extent"}
	if err := schema.Marshal(buf); err != nil {
		t.Fatalf("schema marshal failed: %v", err)
	}

	buf.Append(appendEncodedBytes(nil, payload))
	beforeChecksum := buf.fragments[0].tdata
	checksum := ChecksumFunc(beforeChecksum)

	if disableChecksum {
		checksum = checksum[:0]
	}
	buf.Append(appendEncodedBytes(nil, checksum))

	return buf.fragments[0].tdata, checksum
}

func appendEncodedBytes(dst, payload []byte) []byte {
	dst = append(dst, byte(Tarraym), byte(Tuint8))
	dst = appendSize4(dst, len(payload) + 2)
	dst = appendSize2(dst, len(payload))
	return append(dst, payload...)
}
