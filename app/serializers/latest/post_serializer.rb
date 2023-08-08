# typed: false
# frozen_string_literal: true

class Latest::PostSerializer < Latest::ApplicationSerializer
  attributes :id, :kind, :published_at, :reposts_count, :stamps_count, :viewer_has_stamped

  one :postable, resource: ->(model) do
    case model
    when Latest::Entities::CommentPost
      Latest::Resources::CommentPost
    when Latest::Entities::Repost
      Latest::Resources::Repost
    else
      fail
    end
  end

  one :profile, resource: Latest::Resources::Profile
end
