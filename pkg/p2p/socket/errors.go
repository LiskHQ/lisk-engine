package socket

// export const INTENTIONAL_DISCONNECT_CODE = 1000;
// export const SEED_PEER_DISCONNECTION_REASON = 'Disconnect from SeedPeer after discovery';
// export const INVALID_CONNECTION_URL_CODE = 4501;
// export const INVALID_CONNECTION_URL_REASON =
// 	'Peer did not provide a valid URL as part of the WebSocket connection';
// export const INVALID_CONNECTION_QUERY_CODE = 4502;
// export const INVALID_CONNECTION_QUERY_REASON =
// 	'Peer did not provide valid query parameters as part of the WebSocket connection';
// export const INVALID_CONNECTION_SELF_CODE = 4101;
// export const INVALID_CONNECTION_SELF_REASON = 'Peer cannot connect to itself';
// export const INCOMPATIBLE_NETWORK_CODE = 4102;
// export const INCOMPATIBLE_NETWORK_REASON = 'Peer networkIdentifieinIDr did not match our own';
// export const INCOMPATIBLE_PROTOCOL_VERSION_CODE = 4103;
// export const INCOMPATIBLE_PROTOCOL_VERSION_REASON = 'Peer has incompatible protocol version';
// export const INCOMPATIBLE_PEER_CODE = 4104;
// export const INCOMPATIBLE_PEER_UNKNOWN_REASON =
// 	'Peer is incompatible with the node for unknown reasons';
// export const INCOMPATIBLE_PEER_INFO_CODE = 4105;
// export const INCOMPATIBLE_PEER_INFO_CODE_REASON = 'Peer has invalid PeerInfo';
// // First case to follow HTTP status codes
// export const FORBIDDEN_CONNECTION = 4403;
// export const FORBIDDEN_CONNECTION_REASON = 'Peer is not allowed to connect';
// export const EVICTED_PEER_CODE = 4418;
// export const DUPLICATE_CONNECTION = 4404;
// export const DUPLICATE_CONNECTION_REASON = 'Peer has a duplicate connection';

// Error represents general socket error.
type Error interface {
	Err() error
}
