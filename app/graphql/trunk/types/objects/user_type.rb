# typed: strict
# frozen_string_literal: true

class Trunk::Types::Objects::UserType < Trunk::Types::Objects::Base
  implements GraphQL::Types::Relay::Node

  field :locale, Trunk::Types::Enums::Locale, null: false
  field :profile, Trunk::Types::Objects::ProfileType, null: true do
    argument :atname, String, required: true
  end

  def locale
    object.locale.upcase
  end

  def profile(atname:)
    object.profiles.find_by(atname:)
  end
end
