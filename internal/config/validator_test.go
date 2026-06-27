package config

import "testing"

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		// valid
		{
			name: "valid config",
			config: Config{
				Port: 8080,
				Routes: []Route{
					{Path: "/api/users", BackendURL: []Backend{{URL: "http://localhost:8081"}}, StripPrefix: true},
					{Path: "/api/orders", BackendURL: []Backend{{URL: "http://localhost:8082"}}, StripPrefix: true},
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
						Path: "/api/users",
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
					{Path: "/api/users", BackendURL: []Backend{{URL: "https://api.example.com"}}, StripPrefix: false},
				},
			},
			wantErr: false,
		},

		// port
		{
			name: "invalid port zero",
			config: Config{
				Port:   0,
				Routes: []Route{{Path: "/api/users", BackendURL: []Backend{{URL: "http://localhost:8081"}}}},
			},
			wantErr: true,
		},
		{
			name: "invalid port too high",
			config: Config{
				Port:   65536,
				Routes: []Route{{Path: "/api/users", BackendURL: []Backend{{URL: "http://localhost:8081"}}}},
			},
			wantErr: true,
		},
		{
			name: "invalid port negative",
			config: Config{
				Port:   -1,
				Routes: []Route{{Path: "/api/users", BackendURL: []Backend{{URL: "http://localhost:8081"}}}},
			},
			wantErr: true,
		},

		// routes
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
					{Path: "/api/users", BackendURL: []Backend{{URL: "http://localhost:8081"}}},
					{Path: "/api/users", BackendURL: []Backend{{URL: "http://localhost:8082"}}},
				},
			},
			wantErr: true,
		},

		// backend URL
		{
			name: "missing scheme",
			config: Config{
				Port:   8080,
				Routes: []Route{{Path: "/api/users", BackendURL: []Backend{{URL: "example.com"}}}},
			},
			wantErr: true,
		},
		{
			name: "invalid scheme",
			config: Config{
				Port:   8080,
				Routes: []Route{{Path: "/api/users", BackendURL: []Backend{{URL: "ftp://example.com"}}}},
			},
			wantErr: true,
		},
		{
			name: "missing host",
			config: Config{
				Port:   8080,
				Routes: []Route{{Path: "/api/users", BackendURL: []Backend{{URL: "http://"}}}},
			},
			wantErr: true,
		},
		{
			name: "empty backend URL in slice",
			config: Config{
				Port:   8080,
				Routes: []Route{{Path: "/api/users", BackendURL: []Backend{{URL: ""}}}},
			},
			wantErr: true,
		},
		{
			name: "empty backend slice",
			config: Config{
				Port:   8080,
				Routes: []Route{{Path: "/api/users", BackendURL: []Backend{}}},
			},
			wantErr: true,
		},
		{
			name: "nil backend slice",
			config: Config{
				Port:   8080,
				Routes: []Route{{Path: "/api/users", BackendURL: nil}},
			},
			wantErr: true,
		},
		{
			name: "one invalid backend in multiple backends",
			config: Config{
				Port: 8080,
				Routes: []Route{
					{
						Path: "/api/users",
						BackendURL: []Backend{
							{URL: "http://localhost:8081"},
							{URL: "ftp://localhost:8082"},
						},
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
