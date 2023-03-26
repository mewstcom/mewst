# typed: strict
# frozen_string_literal: true

class Images::AvatarImageComponent < ApplicationComponent
  sig { params(profile: Profile, width: Integer, alt: String).void }
  def initialize(profile:, width:, alt: "")
    @profile = profile
    @width = width
    @alt = T.let(alt.presence || "@#{@profile.atname}", String)
  end

  private

  sig { returns(String) }
  def avatar_url
    @profile.avatar_url.presence || asset_url("avatar.png")
  end
end
