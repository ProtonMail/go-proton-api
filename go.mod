module github.com/ProtonMail/go-proton-api

go 1.24

toolchain go1.24.2

require (
	github.com/Masterminds/semver/v3 v3.2.0
	github.com/ProtonMail/gluon v0.17.1-0.20230724134000-308be39be96e
	github.com/ProtonMail/go-crypto v1.1.4-proton
	github.com/ProtonMail/go-srp v0.0.7
	github.com/ProtonMail/gopenpgp/v2 v2.8.2-proton
	github.com/PuerkitoBio/goquery v1.8.1
	github.com/bradenaw/juniper v0.12.0
	github.com/emersion/go-message v0.16.0
	github.com/emersion/go-vcard v0.0.0-20230331202150-f3d26859ccd3
	github.com/gin-gonic/gin v1.9.1
	github.com/go-resty/resty/v2 v2.7.0
	github.com/google/uuid v1.3.0
	github.com/sirupsen/logrus v1.9.2
	github.com/stretchr/testify v1.8.3
	github.com/urfave/cli/v2 v2.24.4
	gitlab.com/c0b/go-ordered-json v0.0.0-20201030195603-febf46534d5a
	go.uber.org/goleak v1.2.1
	golang.org/x/exp v0.0.0-20230522175609-2e198f4a06a1
	golang.org/x/net v0.38.0
	golang.org/x/text v0.23.0
	google.golang.org/grpc v1.56.3
	google.golang.org/protobuf v1.33.0
)

require (
	github.com/ProtonMail/bcrypt v0.0.0-20211005172633-e235017c1baf // indirect
	github.com/ProtonMail/go-mime v0.0.0-20230322103455-7d82a3887f2f // indirect
	github.com/andybalholm/cascadia v1.3.2 // indirect
	github.com/bytedance/sonic v1.9.1 // indirect
	github.com/chenzhuoyu/base64x v0.0.0-20221115062448-fe3a3abad311 // indirect
	github.com/cloudflare/circl v1.5.0 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.2 // indirect
	github.com/cronokirby/saferith v0.33.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/emersion/go-textwrapper v0.0.0-20200911093747-65d896831594 // indirect
	github.com/gabriel-vasile/mimetype v1.4.2 // indirect
	github.com/gin-contrib/sse v0.1.0 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.14.0 // indirect
	github.com/goccy/go-json v0.10.2 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/cpuid/v2 v2.2.4 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/leodido/go-urn v1.2.4 // indirect
	github.com/mattn/go-isatty v0.0.19 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/pelletier/go-toml/v2 v2.0.8 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rogpeppe/go-internal v1.10.0 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/ugorji/go/codec v1.2.11 // indirect
	github.com/xrash/smetrics v0.0.0-20201216005158-039620a65673 // indirect
	golang.org/x/arch v0.3.0 // indirect
	golang.org/x/crypto v0.36.0 // indirect
	golang.org/x/sync v0.12.0 // indirect
	golang.org/x/sys v0.31.0 // indirect
	google.golang.org/genproto v0.0.0-20230410155749-daa745c078e1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/go-resty/resty/v2 => github.com/LBeernaertProton/resty/v2 v2.0.0-20231129100320-dddf8030d93a
