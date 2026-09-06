# typed: false
# frozen_string_literal: true

RSpec.describe EmailConfirmationMailer do
  describe "#email_confirmation" do
    def setup_data(locale:)
      email_confirmation = FactoryBot.create(:email_confirmation_record, email: "test@example.com", code: "123456")
      mail = EmailConfirmationMailer.email_confirmation(
        email_confirmation_id: email_confirmation.id,
        locale: locale.serialize
      ).message

      {email_confirmation:, mail:}
    end

    context "ロケールが ja のとき" do
      it "日本語の件名で確認用コードを送ること" do
        setup_data(locale: Locale::Ja) => {email_confirmation:, mail:}

        expect(mail.to).to eq([email_confirmation.email])
        expect(mail[:from].value).to start_with("Mewst <no-reply@")
        expect(mail.subject).to eq("[Mewst] 確認用コード")
        expect(mail.body.decoded).to include(email_confirmation.code)
        expect(mail.body.decoded).to include(email_confirmation.email)
      end
    end

    context "ロケールが en のとき" do
      it "英語の件名で確認用コードを送ること" do
        setup_data(locale: Locale::En) => {email_confirmation:, mail:}

        expect(mail.to).to eq([email_confirmation.email])
        expect(mail[:from].value).to start_with("Mewst <no-reply@")
        expect(mail.subject).to eq("[Mewst] Confirmation code")
        expect(mail.body.decoded).to include(email_confirmation.code)
        expect(mail.body.decoded).to include(email_confirmation.email)
      end
    end

    context "件名に非 ASCII 文字が含まれるとき" do
      # Guards the RFC 2047 encode/decode round-trip. The mail gem performs this
      # encoding, so a regression there would silently corrupt Japanese subjects
      # without any change to our own code.
      #
      # [Ja] RFC 2047 のエンコード/デコードの往復を保証する。このエンコードは mail gem が
      # 行うため、gem 側の回帰は自前のコードを変更しないまま日本語の件名を壊しうる。
      it "ヘッダーが ASCII のみになり、デコードすると元の件名に戻ること" do
        setup_data(locale: Locale::Ja) => {mail:}

        encoded_subject = mail[:subject].encoded.sub(/\ASubject:\s*/, "").strip

        expect(encoded_subject).to satisfy { |value| value.ascii_only? }
        expect(encoded_subject).to include("=?UTF-8?")
        expect(Mail::Encodings.value_decode(encoded_subject)).to eq("[Mewst] 確認用コード")
      end
    end
  end
end
