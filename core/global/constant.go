package global

var (
	GPDatabaseUrl        = EnvString("GP_DATABASEURL", "postgres://postgres:rate@localhost:5431/rate?sslmode=disable")
	GPBootstrapMode      = EnvBool("GP_BOOTSTRAP", true)
	GPConnectionAddress  = EnvString("GP_CONNECTIONADDR", "/ip4/0.0.0.0/tcp/8000")
	GPBootstrapAddress   = EnvString("GP_BOOTSTRAPADDR", "")
	GPMinimumSignerCount = EnvInt("GP_MINIMUMSIGNERCOUNT", 3)
	GPFetchPriceInterval = EnvInt("GP_FETCHPRICEINTERVAL", 60)
)
