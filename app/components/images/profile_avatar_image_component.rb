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

  sig { returns(Profile) }
  attr_reader :profile
  private :profile

  sig { returns(String) }
  attr_reader :alt
  private :alt

  sig { returns(Integer) }
  attr_reader :width
  private :width

  sig { returns(String) }
  private def class_name
    class_list = %w[rounded-full]
    class_list << @class_name if @class_name.present?
    class_list.join(" ")
  end

  sig { returns(String) }
  private def size
    "#{width}x#{width}"
  end
end
