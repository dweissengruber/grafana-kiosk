package main

import (
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"

	"github.com/ilyakaznacheev/cleanenv"

	"github.com/grafana/grafana-kiosk/pkg/initialize"
	"github.com/grafana/grafana-kiosk/pkg/kiosk"
)

// Args command-line parameters
type Args struct {
	AutoFit                 bool
	IgnoreCertificateErrors bool
	IsPlayList              bool
	LXDEEnabled             bool
	LXDEHome                string
	ConfigPath              string
	Mode                    string
	LoginMethod             string
	URL                     string
	Username                string
	Password                string
	OTPSecret               string
}

// ProcessArgs processes and handles CLI arguments
func ProcessArgs(cfg interface{}) Args {
	var a Args

	f := flag.NewFlagSet("grafana-kiosk", flag.ContinueOnError)
	f.StringVar(&a.ConfigPath, "c", "", "Path to configuration file (config.yaml)")
	f.StringVar(&a.LoginMethod, "login-method", "anon", "[anon|local|gcom|keycloak]")
	f.StringVar(&a.Username, "username", "guest", "username")
	f.StringVar(&a.Password, "password", "guest", "password")
	f.StringVar(&a.OTPSecret, "otp-secret", "4S62BZNFXXSZLCRO", "otp-secret")
	f.StringVar(&a.Mode, "kiosk-mode", "full", "Kiosk Display Mode [full|tv|disabled]\nfull = No TOPNAV and No SIDEBAR\ntv = No SIDEBAR\ndisabled = omit option\n")
	f.StringVar(&a.URL, "URL", "https://play.grafana.org", "URL to Grafana server")
	f.BoolVar(&a.IsPlayList, "playlists", false, "URL is a playlist")
	f.BoolVar(&a.AutoFit, "autofit", true, "Fit panels to screen")
	f.BoolVar(&a.LXDEEnabled, "lxde", false, "Initialize LXDE for kiosk mode")
	f.StringVar(&a.LXDEHome, "lxde-home", "/home/pi", "Path to home directory of LXDE user running X Server")
	f.BoolVar(&a.IgnoreCertificateErrors, "ignore-certificate-errors", false, "Ignore SSL/TLS certificate error")

	fu := f.Usage
	f.Usage = func() {
		fu()
		envHelp, _ := cleanenv.GetDescription(cfg, nil)
		fmt.Fprintln(f.Output())
		fmt.Fprintln(f.Output(), envHelp)
	}

	err := f.Parse(os.Args[1:])
	if err != nil {
		os.Exit(-1)
	}
	return a
}

func setEnvironment() {
	// for linux/X display must be set
	var displayEnv = os.Getenv("DISPLAY")
	if displayEnv == "" {
		log.Println("DISPLAY not set, autosetting to :0.0")
		os.Setenv("DISPLAY", ":0.0")
		displayEnv = os.Getenv("DISPLAY")
	}
	log.Println("DISPLAY=", displayEnv)

	var xAuthorityEnv = os.Getenv("XAUTHORITY")
	if xAuthorityEnv == "" {
		log.Println("XAUTHORITY not set, autosetting")
		// use HOME of current user
		var homeEnv = os.Getenv("HOME")
		os.Setenv("XAUTHORITY", homeEnv+"/.Xauthority")
		xAuthorityEnv = os.Getenv("XAUTHORITY")
	}
	log.Println("XAUTHORITY=", xAuthorityEnv)
}

func summary(cfg *kiosk.Config) {
	// general
	log.Println("AutoFit:", cfg.General.AutoFit)
	log.Println("LXDEEnabled:", cfg.General.LXDEEnabled)
	log.Println("LXDEHome:", cfg.General.LXDEHome)
	log.Println("Mode:", cfg.General.Mode)
	// target
	log.Println("URL:", cfg.Target.URL)
	log.Println("LoginMethod:", cfg.Target.LoginMethod)
	log.Println("Username:", cfg.Target.Username)
	log.Println("Password:", "*redacted*")
	log.Println("IgnoreCertificateErrors:", cfg.Target.IgnoreCertificateErrors)
	log.Println("IsPlayList:", cfg.Target.IsPlayList)
}

func main() {
	var cfg kiosk.Config
	// override
	args := ProcessArgs(&cfg)
	// check if config specified
	if args.ConfigPath != "" {
		// read configuration from the file and then override with environment variables
		if err := cleanenv.ReadConfig(args.ConfigPath, &cfg); err != nil {
			log.Println("Error reading config file", err)
			os.Exit(-1)
		} else {
			log.Println("Using config from", args.ConfigPath)
		}
	} else {
		log.Println("No config specified, using environment and args")
		// no config, use environment and args
		if err := cleanenv.ReadEnv(&cfg); err != nil {
			log.Println("Error reading config from environment", err)
		}
		cfg.Target.URL = args.URL
		cfg.Target.LoginMethod = args.LoginMethod
		cfg.Target.Username = args.Username
		cfg.Target.Password = args.Password
		cfg.Target.OTPSecret = args.OTPSecret
		cfg.Target.IgnoreCertificateErrors = args.IgnoreCertificateErrors
		cfg.Target.IsPlayList = args.IsPlayList
		//
		cfg.General.AutoFit = args.AutoFit
		cfg.General.LXDEEnabled = args.LXDEEnabled
		cfg.General.LXDEHome = args.LXDEHome
		cfg.General.Mode = args.Mode
	}
	summary(&cfg)
	// make sure the url has content
	if cfg.Target.URL == "" {
		os.Exit(1)
	}
	// validate url
	_, err := url.ParseRequestURI(cfg.Target.URL)
	if err != nil {
		panic(err)
	}
	summary(&cfg)

	if cfg.General.LXDEEnabled {
		initialize.LXDE(cfg.General.LXDEHome)
	}

	// for linux/X display must be set
	setEnvironment()
	log.Println("method ", cfg.Target.LoginMethod)

	switch cfg.Target.LoginMethod {
	case "keycloak":
		log.Printf("Launching keycloak login kiosk")
		kiosk.GrafanaKioskKeycloak(&cfg)
	case "local":
		log.Printf("Launching local login kiosk")
		kiosk.GrafanaKioskLocal(&cfg)
	case "gcom":
		log.Printf("Launching GCOM login kiosk")
		kiosk.GrafanaKioskGCOM(&cfg)
	default:
		log.Printf("Launching ANON login kiosk")
		kiosk.GrafanaKioskAnonymous(&cfg)
	}
}
