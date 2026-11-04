module captcha-demo

go 1.22

require (
	github.com/golang/freetype v0.0.0-20170609003504-e2365dfdc4a0
	github.com/hicode0101/go-captcha v0.0.0
)

require golang.org/x/image v0.23.0 // indirect

replace github.com/hicode0101/go-captcha => ../../../
