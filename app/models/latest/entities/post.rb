# typed: strict
# frozen_string_literal: true

class Latest::Entities::Post < Latest::Entities::Base
  delegate :id, :kind, :published_at, :reposts_count, :stamps_count, to: :post

  sig { params(post: Post, viewer: Profile).void }
  def initialize(post:, viewer:)
    @post = post
    @viewer = viewer
  end

  sig { returns(T.any(Latest::Entities::CommentPost, Latest::Entities::Repost)) }
  def postable
    if post.kind_comment_post?
      Latest::Entities::CommentPost.new(comment_post: post.comment_post!)
    elsif post.kind_repost?
      Latest::Entities::Repost.new(repost: post.repost!, viewer:)
    else
      fail
    end
  end

  sig { returns(Latest::Entities::Profile) }
  def profile
    Latest::Entities::Profile.new(profile: post.profile!)
  end

  sig { returns(T::Boolean) }
  def viewer_has_stamped
    viewer.stamps.exists?(comment_post: post.original_post.comment_post!)
  end

  sig { returns(Post) }
  attr_reader :post
  private :post

  sig { returns(Profile) }
  attr_reader :viewer
  private :viewer
end
