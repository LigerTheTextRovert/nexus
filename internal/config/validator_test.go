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
					{Path: "/api/users", BackendURL: "http://localhost:8081", StripPrefix: true},
					{Path: "/api/orders", BackendURL: "http://localhost:8082", StripPrefix: true},
				},
			},
			wantErr: false,
		},
		{
			name: "valid config with https",
			config: Config{
				Port: 8080,
				Routes: []Route{
					{Path: "/api/users", BackendURL: "https://api.example.com", StripPrefix: false},
				},
			},
			wantErr: false,
		},

		// port
		{
			name: "invalid port zero",
			config: Config{
				Port:   0,
				Routes: []Route{{Path: "/api/users", BackendURL: "http://localhost:8081"}},
			},
			wantErr: true,
		},
		{
			name: "invalid port too high",
			config: Config{
				Port:   65536,
				Routes: []Route{{Path: "/api/users", BackendURL: "http://localhost:8081"}},
			},
			wantErr: true,
		},
		{
			name: "invalid port negative",
			config: Config{
				Port:   -1,
				Routes: []Route{{Path: "/api/users", BackendURL: "http://localhost:8081"}},
			},
			wantErr: true,
		},

		// routes
		{
			name:    "no routes",
			config:  Config{Port: 8080, Routes: []Route{}},
			wantErr: true,
		},

		// backend URL
		{
			name: "missing scheme",
			config: Config{
				Port:   8080,
				Routes: []Route{{Path: "/api/users", BackendURL: "example.com"}},
			},
			wantErr: true,
		},
		{
			name: "invalid scheme",
			config: Config{
				Port:   8080,
				Routes: []Route{{Path: "/api/users", BackendURL: "ftp://example.com"}},
			},
			wantErr: true,
		},
		{
			name: "missing host",
			config: Config{
				Port:   8080,
				Routes: []Route{{Path: "/api/users", BackendURL: "http://"}},
			},
			wantErr: true,
		},
		{
			name: "empty backend URL",
			config: Config{
				Port:   8080,
				Routes: []Route{{Path: "/api/users", BackendURL: ""}},
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
