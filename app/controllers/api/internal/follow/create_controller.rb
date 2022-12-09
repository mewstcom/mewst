# typed: true
# frozen_string_literal: true

class Api::Internal::Follow::CreateController < ApplicationController
  include Authenticatable

  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    target_profile = Profile.only_kept.find_by!(idname: params[:idname])

    if T.must(current_profile).following?(target_profile:)
      return render(json: {}, status: :ok)
    end

    follow_creator = T.must(current_profile).follow(target_profile:)

    if follow_creator.valid?
      return render(json: {}, status: :created)
    end

    render(json: {errors: follow_creator.errors.full_messages}, status: :unprocessable_entity)
  end
end
