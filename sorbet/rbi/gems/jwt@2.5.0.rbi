# typed: true

# DO NOT EDIT MANUALLY
# This is an autogenerated file for types exported from the `jwt` gem.
# Please instead update this file by running `bin/tapioca gem jwt`.

module JWT
  extend ::JWT::Configuration

  private

  def decode(jwt, key = T.unsafe(nil), verify = T.unsafe(nil), options = T.unsafe(nil), &keyfinder); end
  def encode(payload, key, algorithm = T.unsafe(nil), header_fields = T.unsafe(nil)); end

  class << self
    def decode(jwt, key = T.unsafe(nil), verify = T.unsafe(nil), options = T.unsafe(nil), &keyfinder); end
    def encode(payload, key, algorithm = T.unsafe(nil), header_fields = T.unsafe(nil)); end
    def gem_version; end
    def openssl_3?; end
  end
end

module JWT::Algos
  extend ::JWT::Algos

  def find(algorithm); end

  private

  def indexed; end
end

JWT::Algos::ALGOS = T.let(T.unsafe(nil), Array)

module JWT::Algos::Ecdsa
  private

  def curve_by_name(name); end
  def sign(to_sign); end
  def verify(to_verify); end

  class << self
    def curve_by_name(name); end
    def sign(to_sign); end
    def verify(to_verify); end
  end
end

JWT::Algos::Ecdsa::NAMED_CURVES = T.let(T.unsafe(nil), Hash)
JWT::Algos::Ecdsa::SUPPORTED = T.let(T.unsafe(nil), Array)

module JWT::Algos::Eddsa
  private

  def sign(to_sign); end
  def verify(to_verify); end

  class << self
    def sign(to_sign); end
    def verify(to_verify); end
  end
end

JWT::Algos::Eddsa::SUPPORTED = T.let(T.unsafe(nil), Array)

module JWT::Algos::Hmac
  private

  def sign(to_sign); end
  def verify(to_verify); end

  class << self
    def sign(to_sign); end
    def verify(to_verify); end
  end
end

JWT::Algos::Hmac::SUPPORTED = T.let(T.unsafe(nil), Array)

module JWT::Algos::None
  private

  def sign(*_arg0); end
  def verify(*_arg0); end

  class << self
    def sign(*_arg0); end
    def verify(*_arg0); end
  end
end

JWT::Algos::None::SUPPORTED = T.let(T.unsafe(nil), Array)

module JWT::Algos::Ps
  private

  def require_openssl!; end
  def sign(to_sign); end
  def verify(to_verify); end

  class << self
    def require_openssl!; end
    def sign(to_sign); end
    def verify(to_verify); end
  end
end

JWT::Algos::Ps::SUPPORTED = T.let(T.unsafe(nil), Array)

module JWT::Algos::Rsa
  private

  def sign(to_sign); end
  def verify(to_verify); end

  class << self
    def sign(to_sign); end
    def verify(to_verify); end
  end
end

JWT::Algos::Rsa::SUPPORTED = T.let(T.unsafe(nil), Array)

module JWT::Algos::Unsupported
  private

  def sign(*_arg0); end
  def verify(*_arg0); end

  class << self
    def sign(*_arg0); end
    def verify(*_arg0); end
  end
end

JWT::Algos::Unsupported::SUPPORTED = T.let(T.unsafe(nil), Array)

class JWT::Base64
  class << self
    def url_decode(str); end
    def url_encode(str); end
  end
end

class JWT::ClaimsValidator
  def initialize(payload); end

  def validate!; end

  private

  def validate_is_numeric(claim); end
  def validate_numeric_claims; end
end

JWT::ClaimsValidator::NUMERIC_CLAIMS = T.let(T.unsafe(nil), Array)

module JWT::Configuration
  def configuration; end
  def configure; end
end

class JWT::Configuration::Container
  def initialize; end

  def decode; end
  def decode=(_arg0); end
  def jwk; end
  def jwk=(_arg0); end
  def reset!; end
end

class JWT::Configuration::DecodeConfiguration
  def initialize; end

  def algorithms; end
  def algorithms=(_arg0); end
  def leeway; end
  def leeway=(_arg0); end
  def required_claims; end
  def required_claims=(_arg0); end
  def to_h; end
  def verify_aud; end
  def verify_aud=(_arg0); end
  def verify_expiration; end
  def verify_expiration=(_arg0); end
  def verify_iat; end
  def verify_iat=(_arg0); end
  def verify_iss; end
  def verify_iss=(_arg0); end
  def verify_jti; end
  def verify_jti=(_arg0); end
  def verify_not_before; end
  def verify_not_before=(_arg0); end
  def verify_sub; end
  def verify_sub=(_arg0); end
end

class JWT::Configuration::JwkConfiguration
  def initialize; end

  def kid_generator; end
  def kid_generator=(_arg0); end
  def kid_generator_type=(value); end
end

class JWT::Decode
  def initialize(jwt, key, verify, options, &keyfinder); end

  def decode_segments; end

  private

  def algorithm; end
  def allowed_algorithms; end
  def decode_crypto; end
  def find_key(&keyfinder); end
  def header; end
  def none_algorithm?; end
  def options_includes_algo_in_header?; end
  def parse_and_decode(segment); end
  def payload; end
  def segment_length; end
  def set_key; end
  def signing_input; end
  def validate_segment_count!; end
  def verify_algo; end
  def verify_claims; end
  def verify_signature; end
  def verify_signature_for?(key); end
end

class JWT::DecodeError < ::StandardError; end

class JWT::Encode
  def initialize(options); end

  def segments; end

  private

  def combine(*parts); end
  def encode(data); end
  def encode_header; end
  def encode_payload; end
  def encode_signature; end
  def encoded_header; end
  def encoded_header_and_payload; end
  def encoded_payload; end
  def encoded_signature; end
end

JWT::Encode::ALG_KEY = T.let(T.unsafe(nil), String)
JWT::Encode::ALG_NONE = T.let(T.unsafe(nil), String)
class JWT::EncodeError < ::StandardError; end
class JWT::ExpiredSignature < ::JWT::DecodeError; end
class JWT::ImmatureSignature < ::JWT::DecodeError; end
class JWT::IncorrectAlgorithm < ::JWT::DecodeError; end
class JWT::InvalidAudError < ::JWT::DecodeError; end
class JWT::InvalidIatError < ::JWT::DecodeError; end
class JWT::InvalidIssuerError < ::JWT::DecodeError; end
class JWT::InvalidJtiError < ::JWT::DecodeError; end
class JWT::InvalidPayload < ::JWT::DecodeError; end
class JWT::InvalidSubError < ::JWT::DecodeError; end

class JWT::JSON
  class << self
    def generate(data); end
    def parse(data); end
  end
end

module JWT::JWK
  class << self
    def classes; end
    def create_from(keypair, kid = T.unsafe(nil)); end
    def import(jwk_data); end
    def new(keypair, kid = T.unsafe(nil)); end

    private

    def generate_mappings; end
    def mappings; end
  end
end

class JWT::JWK::EC < ::JWT::JWK::KeyBase
  extend ::Forwardable

  def initialize(keypair, options = T.unsafe(nil)); end

  def export(options = T.unsafe(nil)); end
  def key_digest; end
  def keypair; end
  def members; end
  def private?; end
  def public_key(*args, **_arg1, &block); end

  private

  def append_private_parts(the_hash); end
  def encode_octets(octets); end
  def encode_open_ssl_bn(key_part); end
  def keypair_components(ec_keypair); end

  class << self
    def import(jwk_data); end
    def to_openssl_curve(crv); end

    private

    def decode_octets(jwk_data); end
    def decode_open_ssl_bn(jwk_data); end
    def ec_pkey(jwk_crv, jwk_x, jwk_y, jwk_d); end
    def jwk_attrs(jwk_data, attrs); end
  end
end

JWT::JWK::EC::BINARY = T.let(T.unsafe(nil), Integer)
JWT::JWK::EC::KTY = T.let(T.unsafe(nil), String)
JWT::JWK::EC::KTYS = T.let(T.unsafe(nil), Array)

class JWT::JWK::HMAC < ::JWT::JWK::KeyBase
  def initialize(signing_key, options = T.unsafe(nil)); end

  def export(options = T.unsafe(nil)); end
  def key_digest; end
  def keypair; end
  def members; end
  def private?; end
  def public_key; end
  def signing_key; end

  class << self
    def import(jwk_data); end
  end
end

JWT::JWK::HMAC::KTY = T.let(T.unsafe(nil), String)
JWT::JWK::HMAC::KTYS = T.let(T.unsafe(nil), Array)

class JWT::JWK::KeyBase
  def initialize(options); end

  def kid; end

  private

  def generate_kid; end
  def kid_generator; end

  class << self
    def inherited(klass); end
  end
end

class JWT::JWK::KeyFinder
  def initialize(options); end

  def key_for(kid); end

  private

  def find_key(kid); end
  def jwks; end
  def jwks_keys; end
  def load_keys(opts = T.unsafe(nil)); end
  def reloadable?; end
  def resolve_key(kid); end
end

class JWT::JWK::KidAsKeyDigest
  def initialize(jwk); end

  def generate; end
end

class JWT::JWK::RSA < ::JWT::JWK::KeyBase
  def initialize(keypair, options = T.unsafe(nil)); end

  def export(options = T.unsafe(nil)); end
  def key_digest; end
  def keypair; end
  def members; end
  def private?; end
  def public_key; end

  private

  def append_private_parts(the_hash); end
  def encode_open_ssl_bn(key_part); end

  class << self
    def import(jwk_data); end

    private

    def create_rsa_key(rsa_parameters); end
    def decode_open_ssl_bn(jwk_data); end
    def jwk_attributes(jwk_data, *attributes); end
    def rsa_pkey(rsa_parameters); end
  end
end

JWT::JWK::RSA::BINARY = T.let(T.unsafe(nil), Integer)
JWT::JWK::RSA::KTY = T.let(T.unsafe(nil), String)
JWT::JWK::RSA::KTYS = T.let(T.unsafe(nil), Array)
JWT::JWK::RSA::RSA_KEY_ELEMENTS = T.let(T.unsafe(nil), Array)

class JWT::JWK::Thumbprint
  def initialize(jwk); end

  def generate; end
  def jwk; end
  def to_s; end
end

class JWT::JWKError < ::JWT::DecodeError; end
class JWT::MissingRequiredClaim < ::JWT::DecodeError; end
class JWT::RequiredDependencyError < ::StandardError; end

module JWT::SecurityUtils
  private

  def asn1_to_raw(signature, public_key); end
  def raw_to_asn1(signature, private_key); end
  def rbnacl_fixup(algorithm, key); end
  def secure_compare(left, right); end
  def verify_ps(algorithm, public_key, signing_input, signature); end
  def verify_rsa(algorithm, public_key, signing_input, signature); end

  class << self
    def asn1_to_raw(signature, public_key); end
    def raw_to_asn1(signature, private_key); end
    def rbnacl_fixup(algorithm, key); end
    def secure_compare(left, right); end
    def verify_ps(algorithm, public_key, signing_input, signature); end
    def verify_rsa(algorithm, public_key, signing_input, signature); end
  end
end

module JWT::Signature
  private

  def sign(algorithm, msg, key); end
  def verify(algorithm, key, signing_input, signature); end

  class << self
    def sign(algorithm, msg, key); end
    def verify(algorithm, key, signing_input, signature); end
  end
end

class JWT::Signature::ToSign < ::Struct
  def algorithm; end
  def algorithm=(_); end
  def key; end
  def key=(_); end
  def msg; end
  def msg=(_); end

  class << self
    def [](*_arg0); end
    def inspect; end
    def keyword_init?; end
    def members; end
    def new(*_arg0); end
  end
end

class JWT::Signature::ToVerify < ::Struct
  def algorithm; end
  def algorithm=(_); end
  def public_key; end
  def public_key=(_); end
  def signature; end
  def signature=(_); end
  def signing_input; end
  def signing_input=(_); end

  class << self
    def [](*_arg0); end
    def inspect; end
    def keyword_init?; end
    def members; end
    def new(*_arg0); end
  end
end

class JWT::UnsupportedEcdsaCurve < ::JWT::IncorrectAlgorithm; end
module JWT::VERSION; end
JWT::VERSION::MAJOR = T.let(T.unsafe(nil), Integer)
JWT::VERSION::MINOR = T.let(T.unsafe(nil), Integer)
JWT::VERSION::STRING = T.let(T.unsafe(nil), String)
JWT::VERSION::TINY = T.let(T.unsafe(nil), Integer)
class JWT::VerificationError < ::JWT::DecodeError; end

class JWT::Verify
  def initialize(payload, options); end

  def verify_aud; end
  def verify_expiration; end
  def verify_iat; end
  def verify_iss; end
  def verify_jti; end
  def verify_not_before; end
  def verify_required_claims; end
  def verify_sub; end

  private

  def exp_leeway; end
  def global_leeway; end
  def nbf_leeway; end

  class << self
    def verify_aud(payload, options); end
    def verify_claims(payload, options); end
    def verify_expiration(payload, options); end
    def verify_iat(payload, options); end
    def verify_iss(payload, options); end
    def verify_jti(payload, options); end
    def verify_not_before(payload, options); end
    def verify_required_claims(payload, options); end
    def verify_sub(payload, options); end
  end
end

JWT::Verify::DEFAULTS = T.let(T.unsafe(nil), Hash)

class JWT::X5cKeyFinder
  def initialize(root_certificates, crls = T.unsafe(nil)); end

  def from(x5c_header_or_certificates); end

  private

  def build_store(root_certificates, crls); end
  def parse_certificates(x5c_header_or_certificates); end
end
