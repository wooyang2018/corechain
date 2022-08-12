package model

import (
	"testing"

	"github.com/wooyang2018/corechain/ledger"
)

func TestEqual(t *testing.T) {
	testCases := map[string]struct {
		pd     []*ledger.PureData
		vpd    []*ledger.PureData
		expect bool
	}{
		"testEqual": {
			expect: true,
			pd: []*ledger.PureData{
				&ledger.PureData{
					Bucket: "bucket1",
					Key:    []byte("key1"),
					Value:  []byte("value1"),
				},
				&ledger.PureData{
					Bucket: "bucket2",
					Key:    []byte("key2"),
					Value:  []byte("value2"),
				},
			},
			vpd: []*ledger.PureData{
				&ledger.PureData{
					Bucket: "bucket1",
					Key:    []byte("key1"),
					Value:  []byte("value1"),
				},
				&ledger.PureData{
					Bucket: "bucket2",
					Key:    []byte("key2"),
					Value:  []byte("value2"),
				},
			},
		},
		"testNotEqual": {
			expect: false,
			pd: []*ledger.PureData{
				&ledger.PureData{
					Bucket: "bucket1",
					Key:    []byte("key1"),
					Value:  []byte("value1"),
				},
				&ledger.PureData{
					Bucket: "bucket2",
					Key:    []byte("key2"),
					Value:  []byte("value2"),
				},
			},
			vpd: []*ledger.PureData{
				&ledger.PureData{
					Bucket: "bucket1",
					Key:    []byte("key1"),
					Value:  []byte("value1"),
				},
				&ledger.PureData{
					Bucket: "bucket3",
					Key:    []byte("key2"),
					Value:  []byte("value2"),
				},
			},
		},
		"testNotEqual2": {
			expect: false,
			pd: []*ledger.PureData{
				&ledger.PureData{
					Bucket: "bucket1",
					Key:    []byte("key1"),
					Value:  []byte("value1"),
				},
				&ledger.PureData{
					Bucket: "bucket2",
					Key:    []byte("key2"),
					Value:  []byte("value2"),
				},
			},
			vpd: []*ledger.PureData{
				&ledger.PureData{
					Bucket: "bucket1",
					Key:    []byte("key1"),
					Value:  []byte("value1"),
				},
				&ledger.PureData{
					Bucket: "bucket2",
					Key:    []byte("key2"),
					Value:  []byte("value3"),
				},
			},
		},
	}

	for k, v := range testCases {
		res := Equal(v.pd, v.vpd)
		t.Log(res)
		if res != v.expect {
			t.Error(k, "error", "expect", v.expect, "actual", res)
		}
	}
}
