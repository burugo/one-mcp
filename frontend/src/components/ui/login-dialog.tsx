import { useNavigate } from 'react-router-dom';
import { Dialog, DialogContent } from "./dialog";
import { LoginFormCommon } from "./LoginFormCommon";

interface LoginDialogProps {
    isOpen: boolean;
    onClose: () => void;
}

export function LoginDialog({ isOpen, onClose }: LoginDialogProps) {
    const navigate = useNavigate();
    // 登录成功后关闭弹窗并跳转首页
    const handleDialogSuccess = () => {
        onClose();
        navigate('/');
    };
    return (
        <Dialog open={isOpen} onOpenChange={onClose}>
            <DialogContent className="sm:max-w-md p-0 gap-0 border-none bg-background">
                <LoginFormCommon onSuccess={handleDialogSuccess} isDialogMode />
            </DialogContent>
        </Dialog>
    );
} 