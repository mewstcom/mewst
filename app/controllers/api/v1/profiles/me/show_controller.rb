# typed: true
# frozen_string_literal: true

class Api::V1::Profiles::Me::ShowController < Api::V1::ApplicationController
  # include ControllerConcerns::PublicAuthenticatable
  #
  # def call
  #   profile_resource = V1::ProfileResource.new(viewer: current_viewer!, profile: current_viewer!.profile_record.not_nil!)
  #
  #   render(
  #     json: {
  #       profile: V1::ProfileSerializer.new(profile_resource).to_h
  #     }
  #   )
  # end
end
