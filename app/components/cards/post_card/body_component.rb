# typed: strict
# frozen_string_literal: true

class Cards::PostCard::BodyComponent < ApplicationComponent
  sig { params(post: Post, class_name: String).void }
  def initialize(post:, class_name: "")
    @post = post
    @class_name = class_name
  end

  sig { returns(Post) }
  attr_reader :post
  private :post

  sig { returns(String) }
  attr_reader :class_name
  private :class_name

  sig { returns(Profile) }
  private def profile
    post.profile.not_nil!
  end
end
