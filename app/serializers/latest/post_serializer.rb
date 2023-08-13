# typed: false
# frozen_string_literal: true

class Latest::PostSerializer < Latest::ApplicationSerializer
  root_key :post, :posts

  attributes :id, :kind, :published_at, :reposts_count, :stamps_count, :viewer_has_stamped

  one :postable, resource: ->(model) do
    case model
    when Latest::CommentPostResource
      Latest::CommentPostSerializer
    when Latest::RepostResource
      Latest::RepostSerializer
    else
      fail
    end
  end

  one :profile, resource: Latest::ProfileSerializer
end
