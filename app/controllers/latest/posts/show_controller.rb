# typed: true
# frozen_string_literal: true

class Latest::Posts::ShowController < Latest::ApplicationController
  include Latest::Authenticatable

  def call
    post = Post.kept.find_by(id: params[:post_id])

    if post.nil?
      resource = Latest::ResponseErrorResource.new(code: Latest::ResponseErrorCode::NotFound, message: "Not found")
      return render(
        json: Latest::ResponseErrorSerializer.new([resource]),
        status: :not_found
      )
    end

    resource = Latest::PostResource.new(post:, viewer: current_viewer!)
    render(
      json: Latest::PostSerializer.new(resource)
    )
  end
end
