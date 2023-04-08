# typed: strict
# frozen_string_literal: true

class Images::AvatarImageComponent < ApplicationComponent
  sig { params(profile: Profile, width: Integer, alt: String, class_name: String).void }
  def initialize(profile:, width:, alt: "", class_name: "")
    @profile = profile
    @width = width
    @alt = T.let(alt.presence || "@#{@profile.atname}", String)
    @class_name = class_name
  end

  private

  sig { returns(String) }
  def avatar_url
    @profile.avatar_url.presence || asset_url("avatar.png")
  end

  sig { returns(String) }
  def class_name
    class_list = %w[rounded-circle]
    class_list << @class_name if @class_name.present?
    class_list.join(" ")
  end
end
