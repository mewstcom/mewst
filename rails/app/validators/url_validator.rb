# typed: strict
# frozen_string_literal: true

class UrlValidator < ActiveModel::EachValidator
  extend T::Sig

  sig { params(record: ApplicationForm, attribute: Symbol, value: String).void }
  def validate_each(record, attribute, value)
    return if value.blank?

    unless Url.new(value).valid?
      record.errors.add(attribute, :url)
    end
  end
end
