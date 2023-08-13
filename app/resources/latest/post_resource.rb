# typed: strict
# frozen_string_literal: true

class Latest::PostResource < Latest::ApplicationResource
  delegate :id, :kind, :reposts_count, :stamps_count, to: :post

  sig { params(post: Post, viewer: Profile).void }
  def initialize(post:, viewer:)
    @post = post
    @viewer = viewer
  end

  sig { returns(T.any(Latest::CommentPostResource, Latest::RepostResource)) }
  def postable
    if post.kind_comment_post?
      Latest::CommentPostResource.new(comment_post: post.comment_post!)
    elsif post.kind_repost?
      Latest::RepostResource.new(repost: post.repost!, viewer:)
    else
      fail
    end
  end

  sig { returns(Latest::ProfileResource) }
  def profile
    Latest::ProfileResource.new(profile: post.profile!)
  end

  sig { returns(T::Boolean) }
  def viewer_has_stamped
    viewer.stamps.exists?(comment_post: post.original_post.comment_post!)
  end

  sig { returns(String) }
  def published_at
    post.published_at.iso8601
  end

  sig { returns(Post) }
  attr_reader :post
  private :post

  sig { returns(Profile) }
  attr_reader :viewer
  private :viewer
end
