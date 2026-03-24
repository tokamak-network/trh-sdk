package dependencies

import "fmt"

const (
	TerraformVersion = "1.9.8"
	AwsCLIVersion    = "2.22.0"
	KubectlVersion   = "1.31.4"
	HelmVersion      = "3.16.3"

	InstallTimeoutSeconds = 300 // 5 minutes per tool
	MinDiskSpaceMB        = 700
)

func TerraformDownloadURL(arch string) string {
	return fmt.Sprintf("https://releases.hashicorp.com/terraform/%s/terraform_%s_linux_%s.zip", TerraformVersion, TerraformVersion, arch)
}

func AwsCLIDownloadURL(arch string) string {
	a := "x86_64"
	if arch == "arm64" {
		a = "aarch64"
	}
	return fmt.Sprintf("https://awscli.amazonaws.com/awscli-exe-linux-%s-%s.zip", a, AwsCLIVersion)
}

func KubectlDownloadURL(arch string) string {
	return fmt.Sprintf("https://dl.k8s.io/release/v%s/bin/linux/%s/kubectl", KubectlVersion, arch)
}

func KubectlSha256URL(arch string) string {
	return fmt.Sprintf("https://dl.k8s.io/release/v%s/bin/linux/%s/kubectl.sha256", KubectlVersion, arch)
}

func HelmDownloadURL(arch string) string {
	return fmt.Sprintf("https://get.helm.sh/helm-v%s-linux-%s.tar.gz", HelmVersion, arch)
}
