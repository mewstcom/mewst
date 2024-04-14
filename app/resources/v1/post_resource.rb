# typed: strict
# frozen_string_literal: true

class V1::PostResource < V1::ApplicationResource
  delegate :id, :content, to: :post

  sig { params(post: Post, viewer: T.nilable(Actor)).void }
  def initialize(post:, viewer:)
    @post = post
    @viewer = viewer
  end

  sig { returns(V1::ProfileResource) }
  def profile
    V1::ProfileResource.new(profile: post.profile.not_nil!, viewer:)
  end

  sig { returns(V1::ViaResource) }
  def via
    V1::ViaResource.new(oauth_application: post.oauth_application.not_nil!)
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
