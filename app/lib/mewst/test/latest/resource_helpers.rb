# typed: strict
# frozen_string_literal: true

module Mewst::Test::Latest::ResourceHelpers
  extend T::Sig

  sig { params(profile: Profile, viewer_has_followed: T::Boolean).returns(T::Hash[Symbol, T.untyped]) }
  def profile_resource(profile:, viewer_has_followed:)
    {
      id: profile.id,
      atname: profile.atname,
      name: profile.name,
      description: profile.description,
      avatar_url: profile.avatar_url,
      viewer_has_followed:
    }
  end
end
