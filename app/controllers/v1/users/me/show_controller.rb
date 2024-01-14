# typed: true
# frozen_string_literal: true

class V1::Users::Me::ShowController < V1::ApplicationController
  include PublicAuthenticatable

  def call
    user_resource = V1::UserResource.new(user: current_viewer!.user.not_nil!)

    render(
      json: {
        user: V1::UserSerializer.new(user_resource).to_h
      }
    )
  end
end
