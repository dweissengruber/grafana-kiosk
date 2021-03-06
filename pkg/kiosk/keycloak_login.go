package kiosk

import (
	"context"
	"log"
	"time"

	"github.com/xlzd/gotp"

	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/kb"
)

// GrafanaKioskKeycloak creates a chrome-based kiosk using a local grafana-server account
func GrafanaKioskKeycloak(cfg *Config) {
	//dir, err := ioutil.TempDir("", "chromedp-example")
	//if err != nil {
	//	panic(err)
	//}
	//defer os.RemoveAll(dir)

	opts := []chromedp.ExecAllocatorOption{
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
		chromedp.DisableGPU,
		chromedp.Flag("noerrdialogs", true),
		chromedp.Flag("kiosk", true),
		chromedp.Flag("bwsi", true),
		chromedp.Flag("incognito", true),
		chromedp.Flag("disable-sync", true),
		chromedp.Flag("disable-notifications", true),
		chromedp.Flag("disable-overlay-scrollbar", true),
		chromedp.Flag("ignore-certificate-errors", cfg.Target.IgnoreCertificateErrors),
		chromedp.Flag("test-type", cfg.Target.IgnoreCertificateErrors),
		//chromedp.UserDataDir(dir),
		chromedp.UserDataDir("/home/noc/.config/chromium"),
	}

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	// also set up a custom logger
	taskCtx, cancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))
	defer cancel()

	listenChromeEvents(taskCtx, targetCrashed)

	// ensure that the browser process is started
	if err := chromedp.Run(taskCtx); err != nil {
		panic(err)
	}

	var generatedURL = GenerateURL(cfg.Target.URL, cfg.General.Mode, cfg.General.AutoFit, cfg.Target.IsPlayList)
	log.Println("Navigating to ", generatedURL)
	/*
		Launch chrome and login with local user account

		name=user, type=text
		id=inputPassword, type=password, name=password
	*/
	// Give browser time to load next page (this can be prone to failure, explore different options vs sleeping)
	time.Sleep(2000 * time.Millisecond)

	if err := chromedp.Run(taskCtx,
		chromedp.Navigate(generatedURL),
		chromedp.WaitVisible(`//a[@href="login/generic_oauth"]`),
		chromedp.Click(`//a[@href="login/generic_oauth"]`, chromedp.NodeVisible),
		chromedp.WaitVisible(`#kc-header`, chromedp.ByID),
		chromedp.SendKeys(`//input[@name="username"]`, cfg.Target.Username, chromedp.BySearch),
		chromedp.SendKeys(`//input[@name="password"]`, cfg.Target.Password+kb.Enter, chromedp.BySearch),
		chromedp.WaitVisible(`#otp`, chromedp.ByID),
		chromedp.SendKeys(`//input[@name="otp"]`, gotp.NewDefaultTOTP(cfg.Target.OTPSecret).Now()+kb.Enter, chromedp.BySearch),
		chromedp.WaitVisible(`notinputPassword`, chromedp.ByID),
	); err != nil {
		panic(err)
	}
}
