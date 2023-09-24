# typed: true
# frozen_string_literal: true

class Latest::Me::ShowController < Latest::ApplicationController
  include Authenticatable

  def call
    profile_resource = Latest::ProfileResource.new(profile: current_viewer!.profile, viewer: current_viewer!)

    render(
      json: {
        profile: Latest::ProfileSerializer.new(profile_resource).to_h
      }
    )
  end
end
