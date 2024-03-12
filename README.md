# Config

This package Does more than just config at the moment, 

Upon calling `New("config.json")` the code looks for the configuration file and then loads it into memory.  
some other things also happen by doing this:  
* creates a default httpClient
* creates httpClients for any services listed in the config
* sets the hash for the application
* loads the cabundle certs.
* gets and loads the authKeys needed for any of the services listed in the config
* returns all of the above in the Config model object for use in an application.

