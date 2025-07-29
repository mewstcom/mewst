# typed: strict
# frozen_string_literal: true

class UnreservedAtnameValidator < ActiveModel::EachValidator
  extend T::Sig

  sig { params(record: T.any(AccountForm, ProfileRecord), attribute: Symbol, atname: String).void }
  def validate_each(record, attribute, atname)
    return if atname.blank?

    if reserved?(atname)
      record.errors.add(attribute, :reserved)
    end
  end

  sig { params(atname: String).returns(T::Boolean) }
  private def reserved?(atname)
    reserved_atnames = Rails.configuration.mewst["reserved_atnames"]
    reserved_atnames.include?(atname)
  end
end
