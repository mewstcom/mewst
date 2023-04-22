# typed: strict
# frozen_string_literal: true

class Cards::PostCardComponent < ApplicationComponent
  sig { params(post: Post).void }
  def initialize(post:)
    @post = post
  end

  private

  sig { returns(Post) }
  attr_reader :post

  sig { returns(Profile) }
  def profile
    T.must(post.profile)
  end

  def postable_component
    case post.postable_type
    when "CommentedPost"
      Cards::PostCard::CommentedPostContentComponent
    when "CommentedRepost"
      Cards::PostCard::CommentedRepostContentComponent
    when "Repost"
      Cards::PostCard::RepostContentComponent
    else
      fail "Unknown postable type: #{post.postable_type.inspect}"
    end
  end
end
