# typed: strict
# frozen_string_literal: true

module Mewst::Test::Internal::ResourceHelpers
  extend T::Sig

  sig { params(profile: Profile).returns(T::Hash[Symbol, T.untyped]) }
  def profile_resource(profile:)
    {
      id: profile.id,
      atname: profile.atname,
      name: profile.name,
      description: profile.description,
      avatar_url: profile.avatar_url
    }
  end
end
