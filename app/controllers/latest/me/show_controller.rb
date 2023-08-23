# typed: true
# frozen_string_literal: true

class Latest::Me::ShowController < Latest::ApplicationController
  def call
    profile = current_profile.not_nil!
    profile_resource = Latest::ProfileResource.new(profile:, viewer: profile)

    render(
      json: {
        profile: Latest::ProfileSerializer.new(profile_resource).to_h
      }
    )
  end
end
