# typed: false
# frozen_string_literal: true

class Buttons::FollowButtonComponentPreview < ViewComponent::Preview
  def default
    target_profile_entity = ProfileEntity.new(
      atname: "taro",
      name: "山田太郎",
      description: "太郎です",
      avatar_url: "https://example.com/img.jpg",
      viewer_has_followed: false
    )

    render Buttons::FollowButtonComponent.new(target_profile_entity:)
  end
end
