# typed: true
# frozen_string_literal: true

class Api::Internal::Posts::CreateController < ApplicationController
  include Authenticatable

  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    content = T.cast(params[:content], String)
    command = Commands::CreatePost.new(profile: current_profile!, content:)

    if command.invalid?
      return render(json: {errors: command.errors.full_messages}, status: :unprocessable_entity)
    end

    command.call

    @post = T.must(command.post)

    render(status: :created)
  end
end
