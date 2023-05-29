# typed: strict
# frozen_string_literal: true

class Internal::Types::Objects::AccountType < Internal::Types::Objects::Base
  field :locale, Internal::Types::Enums::Locale, null: false

  def locale
    object.locale.upcase
  end
end
