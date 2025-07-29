# typed: strict
# frozen_string_literal: true

class Images::ProfileAvatarImageComponent < ApplicationComponent
  sig { params(profile: ProfileRecord, width: Integer, alt: String, class_name: String).void }
  def initialize(profile:, width:, alt: "", class_name: "")
    @profile = profile
    @width = width
    @alt = T.let(alt.presence || "@#{@profile.atname}", String)
    @class_name = class_name
  end

  sig { returns(ProfileRecord) }
  attr_reader :profile
  private :profile

  sig { returns(Integer) }
  attr_reader :width
  private :width

  sig { returns(String) }
  attr_reader :alt
  private :alt

  sig { returns(String) }
  attr_reader :class_name
  private :class_name
end
