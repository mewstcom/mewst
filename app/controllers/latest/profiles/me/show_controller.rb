# typed: true
# frozen_string_literal: true

class Latest::Profiles::Me::ShowController < Latest::ApplicationController
  include Latest::Authenticatable

  def call
    profile_resource = Latest::ProfileResource.new(viewer: current_viewer!, profile: current_viewer!.profile.not_nil!)

    render(
      json: {
        profile: Latest::ProfileSerializer.new(profile_resource).to_h
      }
    )
  end
end
