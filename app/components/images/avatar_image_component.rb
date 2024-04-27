# typed: strict
# frozen_string_literal: true

class Images::AvatarImageComponent < ApplicationComponent
  sig { params(image_url: String, width: Integer, alt: String, class_name: String).void }
  def initialize(image_url:, width:, alt: "", class_name: "")
    @image_url = image_url
    @width = width
    @alt = alt
    @class_name = class_name
  end

  sig { returns(String) }
  attr_reader :image_url
  private :image_url

  sig { returns(Integer) }
  attr_reader :width
  private :width

  sig { returns(String) }
  attr_reader :alt
  private :alt

  sig { returns(String) }
  attr_reader :class_name
  private :class_name

  sig { returns(String) }
  private def size
    "#{width}x#{width}"
  end
end
