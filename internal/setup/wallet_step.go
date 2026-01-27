package setup

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/yolodolo42/clifi/internal/wallet"
)

// createWallet creates a new wallet with the entered password
func (m WizardModel) createWallet() tea.Cmd {
	password := m.passwordInput.Value()

	return func() tea.Msg {
		km, err := wallet.NewKeystoreManager(m.dataDir)
		if err != nil {
			return walletCreatedMsg{err: err}
		}

		account, err := km.CreateAccount(password)
		if err != nil {
			return walletCreatedMsg{err: err}
		}

		return walletCreatedMsg{address: account.Address.Hex()}
	}
}
