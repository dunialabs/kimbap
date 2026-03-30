package main

import (
	"reflect"
	"testing"
)

func TestRewriteArgsForConfiguredExecutableAlias_CommandAliasBinary(t *testing.T) {
	in := []string{"/usr/local/bin/geosearch", "--name", "San Francisco"}
	aliases := map[string]string{"geosearch": "open-meteo-geocoding.search"}
	got := rewriteArgsForConfiguredExecutableAlias(in, aliases)
	want := []string{"/usr/local/bin/geosearch", "call", "open-meteo-geocoding.search", "--name", "San Francisco"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("rewrite mismatch\nwant=%v\ngot =%v", want, got)
	}
}

func TestRewriteArgsForConfiguredExecutableAlias_KimbapBinaryUnchanged(t *testing.T) {
	in := []string{"/usr/local/bin/kimbap", "call", "open-meteo-geocoding.search"}
	aliases := map[string]string{"geosearch": "open-meteo-geocoding.search"}
	got := rewriteArgsForConfiguredExecutableAlias(in, aliases)
	if !reflect.DeepEqual(got, in) {
		t.Fatalf("expected kimbap binary args unchanged\nwant=%v\ngot =%v", in, got)
	}
}

func TestRewriteArgsForConfiguredExecutableAlias_InvalidTargetUnchanged(t *testing.T) {
	in := []string{"/usr/local/bin/geosearch", "--name", "San Francisco"}
	aliases := map[string]string{"geosearch": "open-meteo-geocoding"}
	got := rewriteArgsForConfiguredExecutableAlias(in, aliases)
	if !reflect.DeepEqual(got, in) {
		t.Fatalf("expected invalid target alias to be ignored\nwant=%v\ngot =%v", in, got)
	}
}
