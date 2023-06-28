# typed: strict
# frozen_string_literal: true

class Trunk::Types::Objects::UserType < Trunk::Types::Objects::Base
  implements GraphQL::Types::Relay::Node

  field :locale, Trunk::Types::Enums::Locale, null: false

  def locale
    object.locale.upcase
  end
end
