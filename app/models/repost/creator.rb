# typed: strict
# frozen_string_literal: true

class Repost::Creator
  extend T::Sig

  include ActiveModel::Model

  sig { returns(Profile) }
  attr_accessor :profile

  sig { returns(String) }
  attr_accessor :post_id

  sig { returns(Post) }
  def call
    target_post = Post.find(post_id)
    post = profile.create_repost(target_post:)
    post
  end
end
