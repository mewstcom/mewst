# typed: strict
# frozen_string_literal: true

class Gravatar
  extend T::Sig

  sig { params(email: String).void }
  def initialize(email:)
    @email = email
  end

  sig { params(size: Integer).returns(String) }
  def url(size:)
    return "" if email.blank?

    params = URI.encode_www_form("s" => size)
    "https://www.gravatar.com/avatar/#{hash}?#{params}"
  end

  sig { returns(String) }
  private def hash
    Digest::SHA256.hexdigest(email)
  end

  sig { returns(String) }
  attr_reader :email
  private :email
end
