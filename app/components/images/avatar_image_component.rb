# typed: strict
# frozen_string_literal: true

class Images::AvatarImageComponent < ApplicationComponent
  sig { params(profile: Profile, width: Integer, alt: String).void }
  def initialize(profile:, width:, alt: "")
    @profile = profile
    @width = width
    @alt = T.let(alt.presence || "@#{@profile.idname}", String)
  end

  private

  sig { returns(T::Boolean) }
  def render?
    @profile.master_avatar.present?
  end
end
