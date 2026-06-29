package config

import "testing"

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		// -------------------------------------------------------------------------
		// Valid configs
		// -------------------------------------------------------------------------
		{
			name: "valid config",
			config: Config{
				Port: 8080,
				Routes: []Route{
					{
						Path:        "/api/users",
						Methods:     []HTTPMethod{GET, POST},
						BackendURL:  []Backend{{URL: "http://localhost:8081"}},
						StripPrefix: true,
					},
					{
						Path:        "/api/orders",
						Methods:     []HTTPMethod{GET},
						BackendURL:  []Backend{{URL: "http://localhost:8082"}},
						StripPrefix: true,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid config with multiple backends",
			config: Config{
				Port: 8080,
				Routes: []Route{
					{
						Path:    "/api/users",
						Methods: []HTTPMethod{GET},
						BackendURL: []Backend{
							{URL: "http://localhost:8081"},
							{URL: "http://localhost:8083"},
						},
						StripPrefix: true,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid config with https",
			config: Config{
				Port: 8080,
				Routes: []Route{
					{
						Path:        "/api/users",
						Methods:     []HTTPMethod{GET},
						BackendURL:  []Backend{{URL: "https://api.example.com"}},
						StripPrefix: false,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid config with rate limit",
			config: Config{
				Port: 8080,
				Routes: []Route{
					{
						Path:       "/api/users",
						Methods:    []HTTPMethod{GET},
						BackendURL: []Backend{{URL: "http://localhost:8081"}},
						RateLimit:  &RateLimit{Requests: 100, Per: "1m"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid config without rate limit",
			config: Config{
				Port: 8080,
				Routes: []Route{
					{
						Path:       "/api/users",
						Methods:    []HTTPMethod{GET},
						BackendURL: []Backend{{URL: "http://localhost:8081"}},
						RateLimit:  nil,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid config all methods",
			config: Config{
				Port: 8080,
				Routes: []Route{
					{
						Path:       "/api/users",
						Methods:    []HTTPMethod{GET, POST, PUT, DELETE, PATCH},
						BackendURL: []Backend{{URL: "http://localhost:8081"}},
					},
				},
			},
			wantErr: false,
		},

		// -------------------------------------------------------------------------
		// Port
		// -------------------------------------------------------------------------
		{
			name: "invalid port zero",
			config: Config{
				Port: 0,
				Routes: []Route{
					{Path: "/api/users", Methods: []HTTPMethod{GET}, BackendURL: []Backend{{URL: "http://localhost:8081"}}},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid port too high",
			config: Config{
				Port: 65536,
				Routes: []Route{
					{Path: "/api/users", Methods: []HTTPMethod{GET}, BackendURL: []Backend{{URL: "http://localhost:8081"}}},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid port negative",
			config: Config{
				Port: -1,
				Routes: []Route{
					{Path: "/api/users", Methods: []HTTPMethod{GET}, BackendURL: []Backend{{URL: "http://localhost:8081"}}},
				},
			},
			wantErr: true,
		},

		// -------------------------------------------------------------------------
		// Routes
		// -------------------------------------------------------------------------
		{
			name:    "no routes",
			config:  Config{Port: 8080, Routes: []Route{}},
			wantErr: true,
		},
		{
			name: "duplicate route paths",
			config: Config{
				Port: 8080,
				Routes: []Route{
					{Path: "/api/users", Methods: []HTTPMethod{GET}, BackendURL: []Backend{{URL: "http://localhost:8081"}}},
					{Path: "/api/users", Methods: []HTTPMethod{GET}, BackendURL: []Backend{{URL: "http://localhost:8082"}}},
				},
			},
			wantErr: true,
		},

		// -------------------------------------------------------------------------
		// Methods
		// -------------------------------------------------------------------------
		{
			name: "no methods",
			config: Config{
				Port: 8080,
				Routes: []Route{
					{Path: "/api/users", Methods: []HTTPMethod{}, BackendURL: []Backend{{URL: "http://localhost:8081"}}},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid method",
			config: Config{
				Port: 8080,
				Routes: []Route{
					{Path: "/api/users", Methods: []HTTPMethod{"INVALID"}, BackendURL: []Backend{{URL: "http://localhost:8081"}}},
				},
			},
			wantErr: true,
		},
		{
			name: "one invalid method among valid ones",
			config: Config{
				Port: 8080,
				Routes: []Route{
					{Path: "/api/users", Methods: []HTTPMethod{GET, "CONNECT"}, BackendURL: []Backend{{URL: "http://localhost:8081"}}},
				},
			},
			wantErr: true,
		},

		// -------------------------------------------------------------------------
		// Backend URL
		// -------------------------------------------------------------------------
		{
			name: "missing scheme",
			config: Config{
				Port:   8080,
				Routes: []Route{{Path: "/api/users", Methods: []HTTPMethod{GET}, BackendURL: []Backend{{URL: "example.com"}}}},
			},
			wantErr: true,
		},
		{
			name: "invalid scheme",
			config: Config{
				Port:   8080,
				Routes: []Route{{Path: "/api/users", Methods: []HTTPMethod{GET}, BackendURL: []Backend{{URL: "ftp://example.com"}}}},
			},
			wantErr: true,
		},
		{
			name: "missing host",
			config: Config{
				Port:   8080,
				Routes: []Route{{Path: "/api/users", Methods: []HTTPMethod{GET}, BackendURL: []Backend{{URL: "http://"}}}},
			},
			wantErr: true,
		},
		{
			name: "empty backend URL in slice",
			config: Config{
				Port:   8080,
				Routes: []Route{{Path: "/api/users", Methods: []HTTPMethod{GET}, BackendURL: []Backend{{URL: ""}}}},
			},
			wantErr: true,
		},
		{
			name: "empty backend slice",
			config: Config{
				Port:   8080,
				Routes: []Route{{Path: "/api/users", Methods: []HTTPMethod{GET}, BackendURL: []Backend{}}},
			},
			wantErr: true,
		},
		{
			name: "nil backend slice",
			config: Config{
				Port:   8080,
				Routes: []Route{{Path: "/api/users", Methods: []HTTPMethod{GET}, BackendURL: nil}},
			},
			wantErr: true,
		},
		{
			name: "one invalid backend in multiple backends",
			config: Config{
				Port: 8080,
				Routes: []Route{
					{
						Path:    "/api/users",
						Methods: []HTTPMethod{GET},
						BackendURL: []Backend{
							{URL: "http://localhost:8081"},
							{URL: "ftp://localhost:8082"},
						},
					},
				},
			},
			wantErr: true,
		},

		// -------------------------------------------------------------------------
		// Path
		// -------------------------------------------------------------------------
		{
			name: "empty path",
			config: Config{
				Port:   8080,
				Routes: []Route{{Path: "", Methods: []HTTPMethod{GET}, BackendURL: []Backend{{URL: "http://localhost:8081"}}}},
			},
			wantErr: true,
		},
		{
			name: "path without leading slash",
			config: Config{
				Port:   8080,
				Routes: []Route{{Path: "api/users", Methods: []HTTPMethod{GET}, BackendURL: []Backend{{URL: "http://localhost:8081"}}}},
			},
			wantErr: true,
		},

		// -------------------------------------------------------------------------
		// Rate limit
		// -------------------------------------------------------------------------
		{
			name: "rate limit zero requests",
			config: Config{
				Port: 8080,
				Routes: []Route{
					{
						Path:       "/api/users",
						Methods:    []HTTPMethod{GET},
						BackendURL: []Backend{{URL: "http://localhost:8081"}},
						RateLimit:  &RateLimit{Requests: 0, Per: "1m"},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "rate limit negative requests",
			config: Config{
				Port: 8080,
				Routes: []Route{
					{
						Path:       "/api/users",
						Methods:    []HTTPMethod{GET},
						BackendURL: []Backend{{URL: "http://localhost:8081"}},
						RateLimit:  &RateLimit{Requests: -10, Per: "1m"},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "rate limit invalid duration string",
			config: Config{
				Port: 8080,
				Routes: []Route{
					{
						Path:       "/api/users",
						Methods:    []HTTPMethod{GET},
						BackendURL: []Backend{{URL: "http://localhost:8081"}},
						RateLimit:  &RateLimit{Requests: 100, Per: "bad"},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "rate limit zero duration",
			config: Config{
				Port: 8080,
				Routes: []Route{
					{
						Path:       "/api/users",
						Methods:    []HTTPMethod{GET},
						BackendURL: []Backend{{URL: "http://localhost:8081"}},
						RateLimit:  &RateLimit{Requests: 100, Per: "0s"},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "rate limit negative duration",
			config: Config{
				Port: 8080,
				Routes: []Route{
					{
						Path:       "/api/users",
						Methods:    []HTTPMethod{GET},
						BackendURL: []Backend{{URL: "http://localhost:8081"}},
						RateLimit:  &RateLimit{Requests: 100, Per: "-1m"},
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
