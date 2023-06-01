# typed: strict
# frozen_string_literal: true

class Internal::Types::Objects::UserType < Internal::Types::Objects::Base
  implements GraphQL::Types::Relay::Node

  field :locale, Internal::Types::Enums::Locale, null: false

  def locale
    object.locale.upcase
  end
end
