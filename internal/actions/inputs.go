package actions

import "os"

type Inputs struct {
	PloiToken    string
	ServerID     string
	Domain       string
	Branch       string
	ProjectType  string
	SystemUser   string
	WebDirectory string
	ProjectRoot  string
	DeployScript string
	Environment  string
	DaemonsYAML  string
	CreateDB     string
	DBEngine     string
	DBName       string
	DBUser       string
	DBHost       string
	DBPort       string
	GithubToken  string
}

func fromEnv() Inputs {
	get := func(k, d string) string {
		v := os.Getenv(k)
		if v == "" {
			return d
		}
		return v
	}
	return Inputs{
		PloiToken:    os.Getenv("PLOI_TOKEN"),
		ServerID:     os.Getenv("SERVER_ID"),
		Domain:       os.Getenv("DOMAIN"),
		Branch:       get("BRANCH", "main"),
		ProjectType:  os.Getenv("PROJECT_TYPE"),
		SystemUser:   get("SYSTEM_USER", "ploi"),
		WebDirectory: get("WEB_DIRECTORY", "/public"),
		ProjectRoot:  get("PROJECT_ROOT", "/"),
		DeployScript: os.Getenv("DEPLOY_SCRIPT"),
		Environment:  os.Getenv("ENVIRONMENT"),
		DaemonsYAML:  os.Getenv("DAEMONS_YAML"),
		CreateDB:     get("CREATE_DATABASE", "false"),
		DBEngine:     get("DB_ENGINE", "mysql"),
		DBName:       os.Getenv("DB_NAME"),
		DBUser:       os.Getenv("DB_USER"),
		DBHost:       get("DB_HOST", "127.0.0.1"),
		DBPort:       get("DB_PORT", "3306"),
		GithubToken:  os.Getenv("GITHUB_TOKEN"),
	}
}
