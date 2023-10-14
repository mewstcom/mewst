# typed: true
# frozen_string_literal: true

class Latest::Users::Me::ShowController < Latest::ApplicationController
  include Latest::Authenticatable

  def call
    user_resource = Latest::UserResource.new(user: current_viewer!.user.not_nil!)

    render(
      json: {
        user: Latest::UserSerializer.new(user_resource).to_h
      }
    )
  end
end
