package memoize

import (
	"math"
	"testing"
)

func TestHashBoolTest(t *testing.T) {
	if hashBool(true) != trueHash {
		t.Errorf("Expected %d, got %d", trueHash, hashBool(true))
	}
	if hashBool(false) != falseHash {
		t.Errorf("Expected %d, got %d", falseHash, hashBool(false))
	}
}

func TestHashStringTest(t *testing.T) {
	expected := hash1("test")
	result := hashString(offset64, "test")
	if result != expected {
		t.Errorf("Expected %d, got %d", expected, result)
	}
}

func TestHashIntTest(t *testing.T) {
	expected := hash1(12345)
	result := hashInt(offset64, 12345)
	if result != expected {
		t.Errorf("Expected %d, got %d", expected, result)
	}
}

func TestHashUintTest(t *testing.T) {
	expected := hash1(12345)
	result := hashUint(offset64, 12345)
	if result != expected {
		t.Errorf("Expected %d, got %d", expected, result)
	}
}

func TestHashFloatTest(t *testing.T) {
	expected := hash1(123.45)
	result := hashFloat(offset64, math.Float64bits(123.45))
	if result != expected {
		t.Errorf("Expected %d, got %d", expected, result)
	}
}

func TestHash1Test(t *testing.T) {
	expected := hashString(offset64, "test")
	result := hash1("test")
	if result != expected {
		t.Errorf("Expected %d, got %d", expected, result)
	}
}

func TestHash2Test(t *testing.T) {
	expected := hashString(hashString(offset64, "test1"), "test2")
	result := hash2("test1", "test2")
	if result != expected {
		t.Errorf("Expected %d, got %d", expected, result)
	}
}

func TestHash3Test(t *testing.T) {
	expected := hashString(hashString(hashString(offset64, "test1"), "test2"), "test3")
	result := hash3("test1", "test2", "test3")
	if result != expected {
		t.Errorf("Expected %d, got %d", expected, result)
	}
}

func TestHash4Test(t *testing.T) {
	expected := hashString(hashString(hashString(hashString(offset64, "test1"), "test2"), "test3"), "test4")
	result := hash4("test1", "test2", "test3", "test4")
	if result != expected {
		t.Errorf("Expected %d, got %d", expected, result)
	}
}

func TestHash5Test(t *testing.T) {
	expected := hashString(hashString(hashString(hashString(hashString(offset64, "test1"), "test2"), "test3"), "test4"), "test5")
	result := hash5("test1", "test2", "test3", "test4", "test5")
	if result != expected {
		t.Errorf("Expected %d, got %d", expected, result)
	}
}

func TestHash6Test(t *testing.T) {
	expected := hashString(hashString(hashString(hashString(hashString(hashString(offset64, "test1"), "test2"), "test3"), "test4"), "test5"), "test6")
	result := hash6("test1", "test2", "test3", "test4", "test5", "test6")
	if result != expected {
		t.Errorf("Expected %d, got %d", expected, result)
	}
}

func TestHash7Test(t *testing.T) {
	expected := hashString(hashString(hashString(hashString(hashString(hashString(hashString(offset64, "test1"), "test2"), "test3"), "test4"), "test5"), "test6"), "test7")
	result := hash7("test1", "test2", "test3", "test4", "test5", "test6", "test7")
	if result != expected {
		t.Errorf("Expected %d, got %d", expected, result)
	}
}
