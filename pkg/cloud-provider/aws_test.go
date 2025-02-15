package cloud_provider

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func Test_LoginAWS(t *testing.T) {
	output, err := LoginAWS("AKIA4MTWNXCGIYG3HPPC", "nXNGf2wzKiOSk/lIg06qM2x6escQaL2JqaAuyAVL", "ap-northeast-2", "json")
	require.Nil(t, err)

	t.Log(output)
}
