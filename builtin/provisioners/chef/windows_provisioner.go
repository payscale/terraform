package chef

import (
	"fmt"
	"path"
	"strings"

	"github.com/hashicorp/terraform/communicator"
	"github.com/hashicorp/terraform/terraform"
)

// there used to be code in this template for version and arch but chef only supports this combo now
const installScript = `
$winver = [System.Environment]::OSVersion.Version | %% {"{0}.{1}" -f $_.Major,$_.Minor}


$url = "https://opscode-omnibus-packages.s3.amazonaws.com/windows/2008r2/x86_64/chef-client-%s-1.msi"
$dest = [System.IO.Path]::GetTempFileName()
$dest = [System.IO.Path]::ChangeExtension($dest, ".msi")
$downloader = New-Object System.Net.WebClient

$http_proxy = '%s'
if ($http_proxy -ne '') {
	$no_proxy = '%s'
  if ($no_proxy -eq ''){
    $no_proxy = "127.0.0.1"
  }

  $proxy = New-Object System.Net.WebProxy($http_proxy, $true, ,$no_proxy.Split(','))
  $downloader.proxy = $proxy
}

Write-Host 'Downloading Chef Client...'
$downloader.DownloadFile($url, $dest)

Write-Host 'Installing Chef Client...'
Start-Process -FilePath msiexec -ArgumentList /qn, /i, $dest -Wait
`

func (p *Provisioner) windowsInstallChefClient(
	o terraform.UIOutput,
	comm communicator.Communicator) error {
	script := path.Join(path.Dir(comm.ScriptPath()), "ChefClient.ps1")
	content := fmt.Sprintf(installScript, p.Version, p.HTTPProxy, strings.Join(p.NOProxy, ","))

	// Copy the script to the new instance
	if err := comm.UploadScript(script, strings.NewReader(content)); err != nil {
		return fmt.Errorf("Uploading client.rb failed: %v", err)
	}

	// Execute the script to install Chef Client
	installCmd := fmt.Sprintf("powershell -NoProfile -ExecutionPolicy Bypass -File %s", script)
	return p.runCommand(o, comm, installCmd)
}

func (p *Provisioner) windowsCreateConfigFiles(
	o terraform.UIOutput,
	comm communicator.Communicator) error {
	// Make sure the config directory exists
	cmd := fmt.Sprintf("cmd /c if not exist %q mkdir %q", windowsConfDir, windowsConfDir)
	if err := p.runCommand(o, comm, cmd); err != nil {
		return err
	}

	if len(p.OhaiHints) > 0 {
		// Make sure the hits directory exists
		hintsDir := path.Join(windowsConfDir, "ohai/hints")
		cmd := fmt.Sprintf("cmd /c if not exist %q mkdir %q", hintsDir, hintsDir)
		if err := p.runCommand(o, comm, cmd); err != nil {
			return err
		}

		if err := p.deployOhaiHints(o, comm, hintsDir); err != nil {
			return err
		}
	}

	return p.deployConfigFiles(o, comm, windowsConfDir)
}
