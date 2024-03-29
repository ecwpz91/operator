package meshconfigs

#secret: {
	_name:    string
	_subject: string
	set_current_client_cert_details?: {...}
	forward_client_cert_details?: string

	secret_validation_name: "spiffe://greymatter.io"
	secret_name:            "spiffe://greymatter.io/\(mesh.metadata.name).\(_name)"
	subject_names: ["spiffe://greymatter.io/\(mesh.metadata.name).\(_subject)"]
	ecdh_curves: ["X25519:P-256:P-521:P-384"]
}

service: {
	clusters: [
		{
			require_tls: true
			secret:      #secret & {
				_name:    "edge"
				_subject: workload.metadata.name
			}
		},
	]

	listener: {
		if workload.metadata.name != "edge" {
			secret: #secret & {
				_name:    workload.metadata.name
				_subject: "edge"
				set_current_client_cert_details: uri: true
				forward_client_cert_details: "APPEND_FORWARD"
			}
		}
	}

	httpEgresses: {
		if len(HTTPEgresses) > 0 {
			clusters: [
				for k, v in HTTPEgresses {
					if v.isExternal {
						{}
					}
					if !v.isExternal {
						{
							require_tls: true
							secret:      #secret & {
								_name:    workload.metadata.name
								_subject: k
							}
						}
					}
				},
			]
		}
	}

	tcpEgresses: [
		for k, v in TCPEgresses {
			{
				clusters: [
					if v.isExternal {
						{}
					},
					if !v.isExternal {
						{
							require_tls: true
							secret:      #secret & {
								_name:    workload.metadata.name
								_subject: k
							}
						}
					},
				]
			}
		},
	]

	localEgresses: [
		for k, v in HTTPEgresses if !v.isExternal {
			k
		},
		for k, v in TCPEgresses if !v.isExternal {
			k
		},
	]
}
