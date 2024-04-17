# typed: strict
# frozen_string_literal: true

class Url
  extend T::Sig

  sig { params(value: String).void }
  def initialize(value)
    @value = value
  end

  sig { returns(T::Boolean) }
  def valid?
    uri.is_a?(Addressable::URI) && !uri.host.nil?
  end

  sig { returns(T.nilable(String)) }
  def host_and_path
    return unless valid?

    "#{uri.host}#{uri.path}"
  end

  sig { params(length: Integer).returns(T.nilable(String)) }
  def shorten_host_and_path(length: 25)
    host_and_path&.truncate(length)
  end

  sig { returns(T.nilable(Addressable::URI)) }
  private def uri
    @uri ||= Addressable::URI.parse(value)
  rescue Addressable::URI::InvalidURIError
    nil
  end

  sig { returns(String) }
  attr_reader :value
  private :value
end
