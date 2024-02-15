# typed: strict
# frozen_string_literal: true

class UrlValidator < ActiveModel::EachValidator
  extend T::Sig

  sig { params(record: ApplicationForm, attribute: Symbol, value: String).void }
  def validate_each(record, attribute, value)
    return if value.blank?

    unless valid_uri?(value)
      record.errors.add(attribute, :url)
    end
  end

  sig { params(value: String).returns(T::Boolean) }
  private def valid_uri?(value)
    uri = Addressable::URI.parse(value)
    uri.is_a?(Addressable::URI) && !uri.host.nil?
  rescue Addressable::URI::InvalidURIError
    false
  end
end
