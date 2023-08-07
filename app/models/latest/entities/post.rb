# typed: strict
# frozen_string_literal: true

class Latest::Entities::Post < Latest::Entities::Base
  delegate :id, :kind, :published_at, :reposts_count, to: :post

  sig { params(post: Post).void }
  def initialize(post:)
    @post = post
  end

  sig { returns(T.any(Latest::Entities::CommentPost, Latest::Entities::Repost)) }
  def postable
    if post.kind_comment_post?
      Latest::Entities::CommentPost.new(comment_post: post.comment_post!)
    elsif post.kind_repost?
      Latest::Entities::Repost.new(repost: post.repost!)
    else
      fail
    end
  end

  sig { returns(Latest::Entities::Profile) }
  def profile
    Latest::Entities::Profile.new(profile: post.profile!)
  end

  sig { returns(Post) }
  attr_reader :post
  private :post
end
