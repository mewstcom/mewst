# typed: strict
# frozen_string_literal: true

class V1::PostResource < V1::ApplicationResource
  delegate :id, :content, to: :post

  sig { params(post: PostRecord, viewer: T.nilable(ActorRecord)).void }
  def initialize(post:, viewer:)
    @post = post
    @viewer = viewer
  end

  sig { returns(V1::ProfileResource) }
  def profile
    V1::ProfileResource.new(profile: post.profile_record.not_nil!, viewer:)
  end

  sig { returns(V1::ViaResource) }
  def via
    V1::ViaResource.new(oauth_application: post.oauth_application.not_nil!)
  end

  sig { returns(T::Boolean) }
  def viewer_has_stamped
    return false if viewer.nil?

    viewer.not_nil!.profile_record.not_nil!.stamp_records.exists?(post_record_id: post.id)
  end

  sig { returns(String) }
  def published_at
    post.published_at.iso8601
  end

  sig { returns(PostRecord) }
  attr_reader :post
  private :post

  sig { returns(T.nilable(ActorRecord)) }
  attr_reader :viewer
  private :viewer
end
