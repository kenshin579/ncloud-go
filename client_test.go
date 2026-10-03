package ncloud

import "testing"

func TestNewClientRequiresBoth(t *testing.T) {
	for _, c := range [][2]string{{"", "s"}, {"i", ""}, {"", ""}} {
		if _, err := NewClient(c[0], c[1]); err == nil {
			t.Errorf("%v: want error", c)
		}
	}
}

func TestNewClientFromEnv(t *testing.T) {
	t.Setenv(EnvClientID, "id")
	t.Setenv(EnvClientSecret, "")
	if _, err := NewClientFromEnv(); err == nil {
		t.Fatal("want error when secret is empty")
	}
	t.Setenv(EnvClientSecret, "sec")
	c, err := NewClientFromEnv()
	if err != nil || c.id != "id" || c.secret != "sec" || c.baseURL != DefaultBaseURL {
		t.Fatalf("got %+v, %v", c, err)
	}
}
