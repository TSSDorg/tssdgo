package tssd

import (
	"errors"
	"testing"
)

func unmarshalFragment(t *testing.T, rbuf *RBuffer, data []byte) (*Fragment, int, error) {
	more, extractErr := rbuf.Extract(data)
	if extractErr != nil || more != 0 {
		return nil, more, extractErr
	}
	return rbuf.Fragment(), 0, nil
}


func TestFragmentUnmarshalSuccess(t *testing.T) {
	payload := []byte("hello fragment")
	data, expectedChecksum := buildFragmentBytes(t, payload, false)

	//we add someting in head, which should drop by TSSD
	data = append(append(make([]byte, 0, 1024), []byte("something")...), data...)

	var rbuf = NewRBuffer()
	frag, more, err :=unmarshalFragment(t, rbuf, append(data, []byte("extra")...))
	if err != nil || more != 0 {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if string(rbuf.data) != "extra" {
		t.Fatalf("expected remaining bytes %q, got %q", "extra", string(rbuf.data))
	}

	if string(frag.Header.Magic[:]) != MAGIC {
		t.Fatalf("expected magic %q, got %q", MAGIC, frag.Header.Magic)
	}
	if frag.Header.Version[0] != TSSD_VERSION_MINOR || frag.Header.Version[1] != TSSD_VERSION_MAJOR {
		t.Fatalf("unexpected version bytes: %v", frag.Header.Version)
	}
	if frag.Schema.FID != 1 {
		t.Fatalf("expected fragment id 1, got %d", frag.Schema.FID)
	}
	if frag.Schema.Types != "hash" || frag.Schema.TID != "tid" || frag.Schema.Info != "extent" {
		t.Fatalf("unexpected schema: %+v", frag.Schema)
	}
	if string(frag.payload) != string(payload) {
		t.Fatalf("expected payload %q, got %q", payload, frag.payload)
	}

	if string(frag.Checksum()) != string(expectedChecksum) {
		t.Fatalf("expected checksum %q, got %q", expectedChecksum, frag.Checksum())
	}

	if string(frag.Payload()) != string(payload) {
		t.Fatalf("expected payload %q, got %q", payload, frag.payload)
	}

	if len(frag.checksum) != 8 + len(expectedChecksum) {
		t.Fatalf("expected checksum %q, got %q", expectedChecksum, frag.Checksum())
	}
}

func TestFragmentUnmarshalRejectsShortInput(t *testing.T) {
	var rbuf = NewRBuffer()
	frag, more, err :=unmarshalFragment(t, rbuf, []byte(MAGIC))

	if !errors.Is(err, ErrorInSufficientData) || more == 0  || frag != nil {
		t.Fatalf("expected ErrorInSufficientData, got %v %v %v", err, more, frag)
	}
}

func TestFragmentUnmarshalRejectsInvalidMagic(t *testing.T) {
	data, _ := buildFragmentBytes(t, []byte("payload"), false)
	data[0] = 'X'

	var rbuf = NewRBuffer()
	_, _, err :=unmarshalFragment(t, rbuf, data)
	if !errors.Is(err, ErrorInSufficientData) {
		t.Fatalf("expected ErrorInSufficientData, got %v", err)
	}
}

func TestFragmentUnmarshalRejectsChecksumMismatch(t *testing.T) {
	data, _ := buildFragmentBytes(t, []byte("payload"), false)
	data[len(data)-1] ^= 1

	var rbuf = NewRBuffer()
	_, _, err :=unmarshalFragment(t, rbuf, data)
	if !errors.Is(err, ErrorInSufficientData) {
		t.Fatalf("expected ErrorInSufficientData, got %v", err)
	}
}

func TestFragmentUnmarshalDisableChecksum(t *testing.T) {
	data, _ := buildFragmentBytes(t, []byte("payload"), true)
	data[len(data)-9] ^= 1

	var rbuf = NewRBuffer()
	frag, _, err :=unmarshalFragment(t, rbuf, data)

	if err != nil || frag == nil {
		t.Fatalf("disableChecksum but got ErrorTSSDDataChecksumFailure")
	}
}

func buildFragmentBytes(t *testing.T, payload []byte, disableChecksum bool) ([]byte, []byte) {
	t.Helper()

	buf := &Buffer{MTU: 4096}
	buf.Append([]byte(MAGIC))
	buf.Append([]byte{TSSD_VERSION_MINOR, TSSD_VERSION_MAJOR, Tschema})

	schema := Schema{FID: 1, Types: "hash", TID: "tid", Info: "extent"}
	if err := schema.marshal(buf); err != nil {
		t.Fatalf("schema marshal failed: %v", err)
	}

	buf.Append(appendEncodedBytes(nil, payload))
	beforeChecksum := buf.fragments[0].payload
	checksum := ChecksumFunc(beforeChecksum)

	if disableChecksum {
		checksum = checksum[:0]
	}
	buf.Append(appendEncodedBytes(nil, checksum))

	return buf.fragments[0].payload, checksum
}

func appendEncodedBytes(dst, payload []byte) []byte {
	dst = append(dst, byte(Tarraym), byte(Tuint8))
	dst = appendSize4(dst, len(payload) + 2)
	dst = appendSize2(dst, len(payload))
	return append(dst, payload...)
}
