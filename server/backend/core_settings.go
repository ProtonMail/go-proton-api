package backend

import (
	"github.com/ProtonMail/go-proton-api"
)

func newUserSettings() proton.UserSettings {
	return proton.UserSettings{Telemetry: proton.SettingEnabled, CrashReports: proton.SettingEnabled}
}
