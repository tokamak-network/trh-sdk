package commands

import (
	"fmt"
	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

type ThanosStack struct {
	Network string
}

func NewThanosStack(network string) *ThanosStack {
	return &ThanosStack{Network: network}
}

func (t *ThanosStack) Deploy() error {
	if !constants.SupportedNetworks[t.Network] {
		return fmt.Errorf("Thanos Stack doesn't support this network: %s", t.Network)
	}
	switch t.Network {
	case constants.LocalDevnet:
		return t.deployLocalDevnet()
	}
	return nil
}

func (t *ThanosStack) deployLocalDevnet() error {
	// install the dependencies for making the devnet up
	doneCh := make(chan bool)

	existingSourcecode, err := utils.CheckExistingSourceCode("tokamak-thanos")
	if err != nil {
		fmt.Println("Error checking existing source code")
		return err
	}

	if !existingSourcecode {
		go utils.ShowLoadingAnimation(doneCh, "Cloning the Thanos Stack repository...")
		err := utils.CloneRepo("https://github.com/tokamak-network/tokamak-thanos.git", "tokamak-thanos")
		doneCh <- true
		if err != nil {
			fmt.Println("Error cloning the repo")
			return err
		}
	}
	fmt.Print("\r✅ Clone the Thanos Stack repository successfully!       \n")

	go utils.ShowLoadingAnimation(doneCh, "Installing the devnet packages...")
	_, err = utils.ExecuteCommand("bash", "-c", "cd tokamak-thanos && bash ./install-devnet-packages.sh")
	if err != nil {
		doneCh <- true
		fmt.Print("\r❌ Installation failed!       \n")
		return err
	}
	fmt.Print("\r✅ Installation completed!       \n")
	doneCh <- true

	go utils.ShowLoadingAnimation(doneCh, "Making the devnet up...")
	output, err := utils.ExecuteCommand("bash", "-c", "cd tokamak-thanos && make devnet-up")
	if err != nil {
		doneCh <- true
		fmt.Printf("\r❌ Make devnet failed!       \n Detail: %s", output)

		return err
	}

	fmt.Print("\r✅ Devnet up!       \n")

	return nil
}
