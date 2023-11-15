# typed: strict
# frozen_string_literal: true

class Latest::PostResource < Latest::ApplicationResource
  delegate :id, :content, :stamps_count, to: :post

  sig { params(post: Post, viewer: T.nilable(Actor)).void }
  def initialize(post:, viewer:)
    @post = post
    @viewer = viewer
  end

  sig { returns(Latest::ProfileResource) }
  def profile
    Latest::ProfileResource.new(profile: post.profile.not_nil!, viewer:)
  end

  sig { returns(Latest::ViaResource) }
  def via
    Latest::ViaResource.new(oauth_application: post.oauth_application.not_nil!)
  end

  sig { returns(T::Boolean) }
  def viewer_has_stamped
    return false if viewer.nil?

    viewer.not_nil!.stamps.exists?(post:)
  end

  sig { returns(String) }
  def published_at
    post.published_at.iso8601
  end

  sig { returns(Post) }
  attr_reader :post
  private :post

  sig { returns(T.nilable(Actor)) }
  attr_reader :viewer
  private :viewer
end
